package grpc

import (
	"encoding/json"
	"fmt"
	"grpc/codec"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
)

const MagicNumber = 0x3bef5c

/*
客户端与服务端的通信需要协商一些内容，
例如 HTTP 报文，分为 header 和 body 2 部分，
body 的格式和长度通过 header 中的 Content-Type 和 Content-Length 指定，
服务端通过解析 header 就能够知道如何从 body 中读取需要的信息。
对于 RPC 协议来说，这部分协商是需要自主设计的。
为了提升性能，一般在报文的最开始会规划固定的字节，来协商相关的信息。
比如第1个字节用来表示序列化方式，第2个字节表示压缩方式，第3-6字节表示 header 的长度，7-10 字节表示 body 的长度。

对于 gRPC 来说，目前需要协商的唯一一项内容是消息的编解码方式。
*/
type Option struct {
	MagicNumber int
	CodecType   codec.Type
}

var DefaultOption = &Option{
	MagicNumber: MagicNumber,   // 标识这是一个 rpc 请求
	CodecType:   codec.GobType, // 编码类型
}

// RPC Server 服务
type Server struct{}

func NewServer() *Server {
	return &Server{}
}

var DefaultServer = NewServer()

// Accept 用于获取连接，并服务请求
func (server *Server) Accept(lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println("rpc server: accept error: ", err)
			return
		}
		// 协程并发处理请求
		go server.ServeConn(conn)
	}
}

// 默认的接收请求并服务
func Accept(lis net.Listener) {
	DefaultServer.Accept(lis)
}

// 为单个连接服务，服务连接直到客户端挂起
func (server *Server) ServeConn(conn io.ReadWriteCloser) {
	defer func() {
		_ = conn.Close()
	}()
	var opt Option

	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("rpc server: option error", err)
		return
	}

	if opt.MagicNumber != MagicNumber {
		log.Println("rpc server: invalid magic number", opt.MagicNumber)
		return
	}

	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		log.Printf("rpc server: invalid codec type %s", opt.CodecType)
		return
	}
	server.serveCodec(f(conn))
}

// 当出现错误时，无效请求作为响应的占位符
var invalidRequest = struct{}{}

func (server *Server) serveCodec(cc codec.Codec) {
	sending := new(sync.Mutex)
	wg := new(sync.WaitGroup)

	for {
		// 获取请求
		req, err := server.readRequest(cc)
		if err != nil {
			if req == nil {
				break
			}
			req.h.Error = err.Error()
			// 回复请求
			server.sendResponse(cc, req.h, invalidRequest, sending)
			continue
		}
		wg.Add(1)
		// 处理请求
		go server.handleRequest(cc, req, sending, wg)
	}

	wg.Wait()
	cc.Close()
}

type request struct {
	h            *codec.Header
	argv, replyv reflect.Value
}

func (server *Server) readRequestHeader(cc codec.Codec) (*codec.Header, error) {
	var h codec.Header
	if err := cc.ReadHeader(&h); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc server: read header error: ", err)
		}
		return nil, err
	}
	return &h, nil
}

// 读取请求
func (server *Server) readRequest(cc codec.Codec) (*request, error) {

	h, err := server.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}

	req := &request{h: h}
	// 目前还不能判断 body 的类型，因此在 readRequest 和 handleRequest 中，day1 将 body 作为字符串处理。
	// 接收到请求，打印 header，并回复 geerpc resp ${req.h.Seq}。
	req.argv = reflect.New(reflect.TypeOf(""))
	if err = cc.ReadBody(req.argv.Interface()); err != nil {
		log.Println("rpc server: read argv err: ", err)
	}
	return req, nil
}

// 回复请求
// 处理请求是并发的，但是回复请求的报文必须是逐个发送的，
// 并发容易导致多个回复报文交织在一起，客户端无法解析。
// 在这里使用锁(sending)保证。
func (server *Server) sendResponse(cc codec.Codec, h *codec.Header, body interface{}, sending *sync.Mutex) {
	sending.Lock()
	defer sending.Unlock()
	if err := cc.Write(h, body); err != nil {
		log.Println("rpc server: write response error: ", err)
	}
}

// 处理请求
func (server *Server) handleRequest(cc codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Println(req.h, req.argv.Elem())
	req.replyv = reflect.ValueOf(fmt.Sprintf("grpc resp: %d", req.h.Seq))
	server.sendResponse(cc, req.h, req.replyv.Interface(), sending)
}
