package geecache

type PeerPicker interface {
	// PickPeer 根据传入的key选择相应节点 PeerGetter
	PickPeer(key string) (peer PeerGetter, ok bool)
}

type PeerGetter interface {
	// Get 用于从对应group中查找缓存值
	Get(group string, key string) ([]byte, error)
}
