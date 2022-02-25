package geecache

import (
	"fmt"
	"log"
	"sync"
)

// Getter 回调getter,当缓存不存在时，应该从数据源中获取数据并添加到缓存中
type Getter interface {
	Get(key string) ([]byte, error)
}

// GetterFunc 将其他函数（参数返回值定义与 F 一致）转换为接口A
// 既能够将普通的函数类型（需类型转换）作为参数，也可以将结构体作为参数，使用更为灵活，可读性也更好，这就是接口型函数的价值。
type GetterFunc func(key string) ([]byte, error)

// Get 函数型结构，实现一个函数，然后在该接口中调用自己
// 传入该接口和直接传入函数的区别是，接口能够被结构体所继承，然后实现在结构体中实现其他的辅助，例如重连，状态维护
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

type Group struct {
	name      string     // 命名空间
	getter    Getter     // 数据未命中时获取数据源的回调
	mainCache cache      // 缓存
	peers     PeerPicker // 节点选择器
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

// RegisterPeers 注册节点
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

// load 从本地或其他节点获取数据源
func (g *Group) load(key string) (value ByteView, err error) {
	// 如果存在节点
	if g.peers != nil {
		// 选择一个节点出来，注意这里可能选到自己，也是返回false,然后执行locally
		if peer, ok := g.peers.PickPeer(key); ok {
			// 从其他节点处获得数据
			if value, err = g.getFromPeer(peer, key); err == nil {
				return value, nil
			}
			log.Println("[MyCache] Failed to get from peer", err)
		}
	}
	return g.getLocally(key)
}

// getFromPeer 从其他节点处获取缓存值
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	// 从节点中获取节点
	bytes, err := peer.Get(g.name, key)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: bytes}, nil
}

// NewGroup 创建一个缓存
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
	}
	groups[name] = g
	return g
}

// GetGroup 获取某缓存
func GetGroup(name string) *Group {
	mu.RLock()
	defer mu.RUnlock()
	g := groups[name]
	return g
}

func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}
	// 从内存缓存中获取，并获取成功
	if v, ok := g.mainCache.get(key); ok {
		log.Println("[MyCache] hit")
		return v, nil
	}
	// 获取失败，从数据源中获取
	return g.load(key)
}

func (g *Group) getLocally(key string) (ByteView, error) {
	// getter是用户传入的数据源获取函数，在这里调用
	bytes, err := g.getter.Get(key)
	// 获取失败，返回空数据
	if err != nil {
		return ByteView{}, err
	}
	// 获取成功，转型
	value := ByteView{b: cloneBytes(bytes)}
	// 写入缓存
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}
