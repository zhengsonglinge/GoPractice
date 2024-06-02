package grpc

import (
	"go/ast"
	"log"
	"reflect"
	"sync/atomic"
)

// 定义方法类型
type methodType struct {
	method    reflect.Method // 方法本身
	ArgType   reflect.Type   // 第一个参数的类型
	ReplyType reflect.Type   // 第二个参数的类型
	numCalls  uint64         // 统计方法调用次数
}

func (m *methodType) NumCalls() uint64 {
	return atomic.LoadUint64(&m.numCalls)
}

// 创建函数参数
func (m *methodType) newArgv() reflect.Value {
	var argv reflect.Value
	// m.ArgType 参数可能是指针类型，也可能是值类型
	if m.ArgType.Kind() == reflect.Ptr {
		argv = reflect.New(m.ArgType.Elem())
	} else {
		argv = reflect.New(m.ArgType).Elem()
	}
	return argv
}

// 创建返回值
func (m *methodType) newReplyv() reflect.Value {
	// 返回值必须是指针类型
	replyv := reflect.New(m.ReplyType.Elem())
	switch m.ReplyType.Elem().Kind() {
	case reflect.Map:
		replyv.Elem().Set(reflect.MakeMap(m.ReplyType.Elem()))
	case reflect.Slice:
		replyv.Elem().Set(reflect.MakeSlice(m.ReplyType.Elem(), 0, 0))
	}
	return replyv
}

// 定义 service
// 用来映射结构体
type service struct {
	name   string                 // 结构体名称
	typ    reflect.Type           // 结构体类型
	rcvr   reflect.Value          // 结构体示例本身，方法调用的接收者，保留 rcvr 是因为在调用时需要 rcvr 作为第 0 个参数
	method map[string]*methodType // 存储结构体可以被远程调用的方法
}

// 根据接收者创建 service，同时将接收者的方法注册到 service 中
func newService(rcvr interface{}) *service {
	s := new(service)
	s.rcvr = reflect.ValueOf(rcvr)
	s.name = reflect.Indirect(s.rcvr).Type().Name()
	s.typ = reflect.TypeOf(rcvr)
	if !ast.IsExported(s.name) {
		log.Fatalf("rpc server: %s is not a valid service name", s.name)
	}
	s.registerMethods()
	return s
}

// 注册方法
func (s *service) registerMethods() {
	s.method = make(map[string]*methodType)
	for i := 0; i < s.typ.NumMethod(); i++ {
		method := s.typ.Method(i)
		mType := method.Type
		// 方法的参数必须是三个，返回值必须是一个
		// 第0个参数是接收者，第1个参数是函数参数，第2个参数是函数返回值
		if mType.NumIn() != 3 || mType.NumOut() != 1 {
			continue
		}
		// 返回值必须是 error
		if mType.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
			continue
		}
		argType, replyType := mType.In(1), mType.In(2)
		// 参数和返回值必须是可导出或内建
		if !isExportedOrBuiltinType(argType) || !isExportedOrBuiltinType(replyType) {
			continue
		}
		s.method[method.Name] = &methodType{
			method:    method,
			ArgType:   argType,
			ReplyType: replyType,
		}
		log.Printf("rpc server: register %s.%s\n", s.name, method.Name)
	}
}

// 判断类型是否是可导出类型或内建类型
func isExportedOrBuiltinType(t reflect.Type) bool {
	return ast.IsExported(t.Name()) || t.PkgPath() == ""
}

// 统计调用次数
// 传入参数调用方法，获取返回值
// 参数的第一个值是调用者即接收者
// 返回值的第一个值是错误
func (s *service) call(m *methodType, argv, replyv reflect.Value) error {
	atomic.AddUint64(&m.numCalls, 1)
	// 获取方法并执行
	f := m.method.Func
	returnValues := f.Call([]reflect.Value{s.rcvr, argv, replyv})
	if errInter := returnValues[0].Interface(); errInter != nil {
		return errInter.(error)
	}
	return nil
}
