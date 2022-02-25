package lru

import "container/list"

type Cache struct {
	maxBytes  int64                         // 允许使用的最大内存
	nBytes    int64                         // 已使用的内存
	ll        *list.List                    // 双向链表
	Cache     map[string]*list.Element      // 缓存字典
	OnEvicted func(key string, value Value) // 条目被清除时的回调
}

// entry 在双向链表里存在的元素，放kv而不是只放v的原因，当需要淘汰元素时，需要把k从map中删除，如果只放v则需要遍历map，现在有了k可以直接删去
type entry struct {
	key   string
	value Value
}

// Value 值需要实现这个结构，并且能够获取值的大小
type Value interface {
	Len() int
}

// New cache初始化函数
func New(maxBytes int64, onEvicted func(string2 string, value Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		Cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

func (c *Cache) Get(key string) (v Value, ok bool) {
	// 获取元素
	if ele, ok := c.Cache[key]; ok {
		// 该元素被读，将其放在队头
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return
}

// RemoveOldest 异常最老的item
func (c *Cache) RemoveOldest() {
	// 取得最后的那个节点
	ele := c.ll.Back()
	if ele == nil {
		return
	}
	// 删除该节点
	c.ll.Remove(ele)
	// 删除该映射关系
	kv := ele.Value.(*entry)
	delete(c.Cache, kv.key)
	// 减少占用空间
	c.nBytes -= int64(len(kv.key)) + int64(kv.value.Len())
	if c.OnEvicted != nil {
		c.OnEvicted(kv.key, kv.value)
	}
}

// Add 加入键值对
func (c Cache) Add(key string, value Value) {
	// 该该键值本来就存在于map中
	if ele, ok := c.Cache[key]; ok {
		// 移到队首
		c.ll.MoveToFront(ele)
		// 刷新
		kv := ele.Value.(*entry)
		// 因为是更新，所以加上新值和旧值的差值
		c.nBytes += int64(value.Len()) - int64(kv.value.Len())
		// 刷新值
		kv.value = value
	} else {
		// 不存在，在队列里加入
		ele = c.ll.PushFront(&entry{key: key, value: value})
		// 建立映射关系
		c.Cache[key] = ele
		// 加上map的key和队列item的size
		c.nBytes += int64(len(key)) + int64(value.Len())
	}
	// 更新占用值，如果超过最大值，移除最少访问的节点
	for c.maxBytes != 0 && c.nBytes > c.maxBytes {
		c.RemoveOldest()
	}
}
