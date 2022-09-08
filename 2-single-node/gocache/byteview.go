package gocache

//	ByteView 保存字节的不变view
type ByteView struct {
	//	存储真实的缓存值 b 是只读的 防止缓存值被外部程序修改
	b []byte
}

//	Len 返回view的长度
func (v ByteView) Len() int {
	return len(v.b)
}

//	ByteSlice 返回数据作为byte切片的拷贝
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

//	String 返回数据作为字符串，若果需要做一个拷贝
func (v ByteView) String() string {
	return string(v.b)
}

func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
