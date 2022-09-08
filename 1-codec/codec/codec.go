package codec

import "io"

//	body 剩余信息接口
type Header struct {
	//	rpc 调用方法名 Service.Method
	ServiceMethod string
	//	rpc 请求序号，用来区分不同的请求
	Seq uint64
	//	错误信息
	Error string
}

//	对消息进行解编码的接口
type Codec interface {
	io.Closer
	ReadHeader(*Header) error
	ReadBody(interface{}) error
	Write(*Header, interface{}) error
}

//	NewCodecFunc 抽象编解码 构造函数
type NewCodecFunc func(closer io.ReadWriteCloser) Codec

//	Type 编解码类型
type Type string

//	定义两个编解码类型
const (
	GobType Type = "application/gob"
	//	JsonType 未实现
	JsonType Type = "application/json"
)

var NewCodecFuncMap map[Type]NewCodecFunc

func init() {
	NewCodecFuncMap = make(map[Type]NewCodecFunc)
	NewCodecFuncMap[GobType] = NewGobCodec
}
