package lru

import "container/list"

//	Cache是lru cache，不是并发安全的
type Cache struct {
	maxBytes  int64                         //	允许使用的最大指针
	nbytes    int64                         //	当前已使用的内存
	ll        *list.List                    //	双向链表
	cache     map[string]*list.Element      //	Element为对应结点的指针
	OnEvicted func(key string, value Value) //	回调函数  当某条记录被移除时
}

//	entry 结点的数据类型
type entry struct {
	key   string
	value Value
}

//	Value 用于返回值所占用的内存大小
type Value interface {
	Len() int
}

//	New 构造Cache
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

//	Get 查询key的value
func (c *Cache) Get(key string) (value Value, ok bool) {
	//	从字典中找到对应的结点
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele) //	将ele移动到链表的第一个位置  这里指队尾
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return
}

//	RemoveOldest 删除最旧的项目
func (c *Cache) RemoveOldest() {
	ele := c.ll.Back() //	返回链表最后一个元素或nil
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		//	从map中删除
		delete(c.cache, kv.key)
		//	更新当前内存
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok { //	存在
		//	将结点移到队尾
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		//	更新对应结点的值
		kv.value = value
	} else { //	不存在
		//	新增场景 添加新节点
		ele := c.ll.PushFront(&entry{key, value})
		c.cache[key] = ele
		c.nbytes += int64(len(key)) + int64(value.Len())
	}
	for c.maxBytes != 0 && c.maxBytes < c.nbytes { //	超过了设定的最大值
		c.RemoveOldest()
	}
}

//	实现 Len()
func (c *Cache) Len() int {
	return c.ll.Len()
}
