package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

//	Hash将字节映射到uint32
//	采取依赖注入的方式，允许用于替换成自定义的Hash函数
type Hash func(data []byte) uint32

//	Map contains all hashed keys
type Map struct {
	hash     Hash
	replicas int            //	虚拟节点倍数
	keys     []int          //	哈希环
	hashMap  map[int]string //	虚拟节点与真实节点的映射表
}

//	New 穿件一个Map实例
func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		//	返回使用IEEE多项式计算出的CRC-32校验和
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

//	Add添加真实节点
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			//	虚拟节点的名称为：strconv.Itoa(i) + key
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			//	添加到环
			m.keys = append(m.keys, hash)
			m.hashMap[hash] = key
		}
	}
	sort.Ints(m.keys)
}

func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}

	hash := int(m.hash([]byte(key)))

	//	顺时针找到第一个匹配的虚拟节点
	//	二进制搜索
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})

	return m.hashMap[m.keys[idx%len(m.keys)]]
}
