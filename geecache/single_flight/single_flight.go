package single_flight

import "sync"

// call 正在进行中或者已经结束的请求
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

type Group struct {
	mu sync.Mutex
	m  map[string]*call
}

// Do 针对相同的 key，无论 Do 被调用多少次，函数 fn 都只会被调用一次
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	// 上锁
	g.mu.Lock()
	// 新建map结构记录
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	// 从map获取请求
	if c, ok := g.m[key]; ok {
		// 如果请求在进行中，解锁，等待
		// 为什么这里要解锁，因为主要是为了保护map的线程安全，如果获取后如果是存在的就需要阻塞等待，固然不能带锁等待
		g.mu.Unlock()
		// 等待信号
		c.wg.Wait()
		return c.val, c.err
	}
	// 没有正在进行中的请求
	c := new(call)
	c.wg.Add(1)
	// 加入请求
	g.m[key] = c
	g.mu.Unlock()
	c.val, c.err = fn()
	c.wg.Done()
	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()
	return c.val, c.err
}
