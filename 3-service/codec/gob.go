package codec

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

type GobCodec struct {
	//	由构建函数传入	socket链接实例
	conn io.ReadWriteCloser
	//	防止阻塞 带缓冲的 Writer
	buf *bufio.Writer
	//	解码/反序列化
	dec *gob.Decoder
	//	编码/序列化
	enc *gob.Encoder
}

//	检查 结构体是否实现接口
var _ Codec = (*GobCodec)(nil)

//	NewGobCodec 构造函数
func NewGobCodec(conn io.ReadWriteCloser) Codec {
	buf := bufio.NewWriter(conn)
	return &GobCodec{
		conn: conn,
		buf:  buf,
		//	解码 -> 读取一次连接中的所有信息
		dec: gob.NewDecoder(conn),
		//	编码 -> 往一个新的buf写缓冲里写入内容
		enc: gob.NewEncoder(buf),
	}
}

//	ReadHeader 获取请求头
func (c *GobCodec) ReadHeader(h *Header) error {
	return c.dec.Decode(h)
}

//	ReadBody 获取请求体
func (c *GobCodec) ReadBody(body interface{}) error {
	return c.dec.Decode(body)
}

func (c *GobCodec) Write(h *Header, body interface{}) (err error) {
	defer func() {
		//	缓冲区写入
		_ = c.buf.Flush()
		//	错误这关闭连接
		if err != nil {
			_ = c.Close()
		}
	}()
	//	请求头 错误处理
	if err := c.enc.Encode(h); err != nil {
		log.Println("rpc code: gob error encoding header: ", err)
		return err
	}
	//	请求体 错误处理
	if err := c.enc.Encode(body); err != nil {
		log.Println("rpc code: gob error encoding body: ", err)
		return err
	}
	return nil
}

//	Close 断开连接
func (c *GobCodec) Close() error {
	return c.conn.Close()
}
