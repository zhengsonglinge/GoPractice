package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"grpc/codec"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

/*
实现一个 RPC 客户端：
1. 创建连接：是通过 Dial 函数调用 NewClient 函数创建客户端，创建过程中发送编码类型
2. 接收请求：创建 client 的同时调用协程 go client.receive() 函数接收处理请求
3. 发送请求：Go 异步调用，Call 同步调用，调用 send 函数发送数据给服务端
*/

// 对 net/rpc 而言，一个函数需能够被远程调用，需要满足如下五个条件：
// the method’s type is exported.
// the method is exported.
// the method has two arguments, both exported (or builtin) types.
// the method’s second argument is a pointer.
// the method has return type error.

// Call 代表一条 RPC 消息
type Call struct {
	Seq           uint64
	ServiceMethod string      // format "<service>.<method>"
	Args          interface{} // arguments to the function
	Reply         interface{} // reply from the function
	Error         error       // if error occurs, it will be set
	// 为了支持异步调用，Call 结构体中添加了一个字段 Done，
	// Done 的类型是 chan *Call，当调用结束时，会调用 call.done() 通知调用方。
	Done chan *Call // Strobes when call is complete.
}

func (call *Call) done() {
	call.Done <- call
}

type Client struct {
	cc       codec.Codec // 编解码器，序列化发送的请求和接收的响应
	opt      *Option
	sending  sync.Mutex       // 互斥锁，和服务端类似，为了保证请求的有序发送，即防止出现多个请求报文混淆
	header   codec.Header     // 每个请求的消息头，header 只有在请求发送时才需要，而请求发送是互斥的，因此每个客户端只需要一个，声明在 Client 结构体中可以复用。
	mu       sync.Mutex       //
	seq      uint64           // 用于给发送的请求编号，每个请求拥有唯一编号。
	pending  map[uint64]*Call // 存储未处理完的请求，键是编号，值是 Call 实例。
	closing  bool             // closing 是用户主动关闭的，即调用 Close 方法
	shutdown bool             // shutdown 置为 true 一般是有错误发生。
	// closing 和 shutdown 任意一个值置为 true，则表示 Client 处于不可用的状态
}

var _ io.Closer = (*Client)(nil)

var ErrShutdown = errors.New("connection is shut down")

// gRPC 在 3 个地方添加了超时处理机制。分别是：
// 1）客户端创建连接时
// 2）客户端 Client.Call() 整个过程导致的超时（包含发送报文，等待处理，接收报文所有阶段）
// 3）服务端处理报文，即 Server.handleRequest 超时。
type clientResult struct {
	client *Client
	err    error
}

type newClientFunc func(conn net.Conn, opt *Option) (client *Client, err error)

// dial 函数添加超时控制的代码
func dialTimeout(f newClientFunc, network, address string, opts ...*Option) (client *Client, err error) {
	opt, err := parseOptions(opts...)
	if err != nil {
		return nil, err
	}

	// 超时连接，如果连接超时则返回错误
	conn, err := net.DialTimeout(network, address, opt.ConnectTimeout)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = conn.Close()
		}
	}()

	// 使用子协程执行 NewClient，执行完成后则通过信道 ch 发送结果，
	// 如果 time.After() 信道先接收到消息，则说明 NewClient 执行超时，返回错误。
	ch := make(chan clientResult)
	go func() {
		client, err := f(conn, opt)
		ch <- clientResult{client: client, err: err}
	}()
	if opt.ConnectTimeout == 0 {
		result := <-ch
		return result.client, result.err
	}
	select {
	case <-time.After(opt.ConnectTimeout):
		return nil, fmt.Errorf("rpc client: connect timeout: expect within %s", opt.ConnectTimeout)
	case result := <-ch:
		return result.client, result.err
	}
}

// 实现 Dial 函数用来传入服务端地址，创建客户端
// Option 参数是可选的
func Dial(network, address string, opts ...*Option) (client *Client, err error) {
	return dialTimeout(NewClient, network, address, opts...)
}

// 解析 Option
func parseOptions(opts ...*Option) (*Option, error) {
	if len(opts) == 0 || opts[0] == nil {
		return DefaultOption, nil
	}
	if len(opts) != 1 {
		return nil, errors.New("number of options is more than 1")
	}
	opt := opts[0]
	opt.MagicNumber = DefaultOption.MagicNumber
	if opt.CodecType == "" {
		opt.CodecType = DefaultOption.CodecType
	}
	return opt, nil
}

// 创建 Client 实例时，首先需要完成一开始的协议交换，即发送 Option 信息给服务端。
// 协商好消息的编解码方式之后，再创建一个子协程调用 receive() 接收响应。
func NewClient(conn net.Conn, opt *Option) (*Client, error) {
	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		err := fmt.Errorf("invalid codec type %s", opt.CodecType)
		log.Println("rpc client: codec error:", err)
		return nil, err
	}
	// 发送编码类型给服务端
	if err := json.NewEncoder(conn).Encode(opt); err != nil {
		log.Println("rpc client: options error: ", err)
		_ = conn.Close()
		return nil, err
	}
	return newClientCodec(f(conn), opt), nil
}

// 创建 client 之后再用一个协程接收请求
func newClientCodec(cc codec.Codec, opt *Option) *Client {
	client := &Client{
		seq:     1, // seq 从 1 开始, 0 标识无效 call
		cc:      cc,
		opt:     opt,
		pending: make(map[uint64]*Call),
	}
	go client.receive()
	return client
}

// 实现客户端的接收功能，有三种情况:
// 1. call 不存在，可能是请求没有发送完整，或者因为其他原因被取消，但是服务端仍旧处理了。
// 2. call 存在，但服务端处理出错，即 h.Error 不为空。
// 3. call 存在，服务端处理正常，那么需要从 body 中读取 Reply 的值。
func (client *Client) receive() {
	var err error
	for err == nil {
		var h codec.Header
		if err = client.cc.ReadHeader(&h); err != nil {
			break
		}
		call := client.removeCall(h.Seq)
		switch {
		// 写入部分失败或 call 已经被处理过
		case call == nil:
			err = client.cc.ReadBody(nil)
		case h.Error != "":
			call.Error = fmt.Errorf(h.Error)
			err = client.cc.ReadBody(nil)
			call.done()
		default:
			err = client.cc.ReadBody(call.Reply)
			if err != nil {
				call.Error = errors.New("reading body " + err.Error())
			}
			call.done()
		}
	}
	// 发生错误，则未处理的 call 全部置为 err 并终止客户端
	client.terminateCalls(err)
}

// Go 和 Call 是客户端暴露给用户的两个 RPC 服务调用接口，Go 是一个异步接口，返回 call 实例。
func (client *Client) Go(serviceMethod string, args, reply interface{}, done chan *Call) *Call {
	if done == nil {
		done = make(chan *Call, 10)
	} else if cap(done) == 0 {
		log.Panic("rpc client: done channel is unbuffered")
	}
	call := &Call{
		ServiceMethod: serviceMethod,
		Args:          args,
		Reply:         reply,
		Done:          done,
	}
	client.send(call)
	return call
}

// Call 是对 Go 的封装，阻塞 call.Done，等待响应返回，是一个同步接口。
// 添加超时控制代码，使用 context 包实现，WithCancel 和 WithTimeout 都可以触发 ctx.Done()
func (client *Client) Call(ctx context.Context, serviceMethod string, args, reply interface{}) error {
	call := client.Go(serviceMethod, args, reply, make(chan *Call, 1))
	select {
	case <-ctx.Done():
		client.removeCall(call.Seq)
		return errors.New("rpc client: call failed: " + ctx.Err().Error())
	case call := <-call.Done:
		return call.Error
	}
}

// 实现客户端的发送请求功能
func (client *Client) send(call *Call) {
	// 加锁确保客户端能发送完整消息
	client.sending.Lock()
	defer client.sending.Unlock()

	// 注册一个 call
	seq, err := client.registerCall(call)
	if err != nil {
		call.Error = err
		call.done()
		return
	}

	// 请求头
	client.header.ServiceMethod = call.ServiceMethod
	client.header.Seq = seq
	client.header.Error = ""

	// 编码并发送数据
	if err := client.cc.Write(&client.header, call.Args); err != nil {
		call := client.removeCall(seq)
		// 当 call 为 nil 时写入部分失败或 call 已经被处理过
		if call != nil {
			call.Error = err
			call.done()
		}
	}
}

// 关闭连接
func (client *Client) Close() error {
	client.mu.Lock()
	defer client.mu.Unlock()
	// 已经被关闭则返回关闭失败
	if client.closing {
		return ErrShutdown
	}
	client.closing = true
	return client.cc.Close()
}

// 判断客户端是否可用
func (client *Client) IsAvailable() bool {
	client.mu.Lock()
	defer client.mu.Unlock()
	return !client.shutdown && !client.closing
}

// 将参数 call 添加到 client.pending 中，并更新 client.seq
func (client *Client) registerCall(call *Call) (uint64, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closing || client.shutdown {
		return 0, ErrShutdown
	}
	call.Seq = client.seq
	client.pending[call.Seq] = call
	client.seq++
	return call.Seq, nil
}

// 根据 seq，从 client.pending 中移除对应的 call
func (client *Client) removeCall(seq uint64) *Call {
	client.mu.Lock()
	defer client.mu.Unlock()
	call := client.pending[seq]
	delete(client.pending, seq)
	return call
}

// 服务端或客户端发生错误时调用，将 shutdown 设置为 true
// 且将错误信息通知所有 pending 状态的 call。
func (client *Client) terminateCalls(err error) {
	client.sending.Lock()
	defer client.sending.Unlock()
	client.mu.Lock()
	defer client.mu.Unlock()
	client.shutdown = true
	for _, call := range client.pending {
		call.Error = err
		call.done()
	}
}
