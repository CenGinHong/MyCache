package geecache

// ByteView 存储真实的缓存值
type ByteView struct {
	b []byte
}

// Len 大小
func (v ByteView) Len() int {
	return len(v.b)
}

// ByteSlice 返回拷贝，防止被外部程序修改
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

func (v ByteView) String() string {
	return string(v.b)
}

func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
