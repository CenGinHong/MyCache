package consistent

import (
	"hash/crc32"
	"sort"
	"strconv"
)

type Hash func(data []byte) uint32

type Map struct {
	hash     Hash
	replicas int            // 虚拟节点倍数
	keys     []int          // 哈希环
	hashMap  map[int]string // 虚拟节点和真实节点的映射表
}

// New 添加真实节点/机器的方法
func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		// 可以理解为哈希函数
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// Add 添加虚拟节点到哈希环上
func (m Map) Add(keys ...string) {
	// keys是多个真实节点的名称
	for _, key := range keys {
		// 对于每一个真实节点key，创建replicas个虚拟节点，加到hash环上
		for i := 0; i < m.replicas; i++ {
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			// 加到hash环
			m.keys = append(m.keys, hash)
			m.hashMap[hash] = key
		}
	}
	// 排序
	sort.Ints(m.keys)
}

// Get 传入缓存k，获取距离该k最近的节点，并映射回虚拟节点
func (m Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}
	// 获取hash值
	hash := int(m.hash([]byte(key)))
	// 顺时针找到第一个下标idx
	// 使用二分查找的方法，会从[0, n)中取出一个值index，index为[0, n)中最小的使函数f(index)为True的值，并且f(index+1)也为True。
	// 一般用于从一个已经排序的数组中找到某个值所对应的索引。
	// 相当于一个三角形，就是找到落点处最近的端点，如果找不到（在往上已经找不到）返回n,由于是环形，所以n对应节点0
	idx := sort.Search(len(m.keys), func(i int) bool {
		return hash <= m.keys[i]
	})
	// 找不到会返回n，所以需要取余一下
	// 如果 idx == len(m.keys)，说明应选择 m.keys[0]，因为 m.keys 是一个环状结构，所以用取余数的方式来处理这种情况。
	return m.hashMap[m.keys[idx%len(m.keys)]]
}
