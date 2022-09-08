package gocache

import (
	"fmt"
	"log"
	"sync"
)

//	Getter 加载key的数据
type Getter interface {
	Get(key string) ([]byte, error)
}

//	GetterFunc 用函数实现Getter
type GetterFunc func(key string) ([]byte, error)

//	Get 实现Getter接口函数	回调函数
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

//	Group是一个cache名称空间和相关的数据被分散加载  缓存的命名空间
type Group struct {
	name      string //	唯一的名称
	getter    Getter //	缓存未命中时获取源数据的回调
	mainCache cache  //	一开始实现的并发缓存
	peers     PeerPicker
}

var (
	mu     sync.Mutex
	groups = make(map[string]*Group)
)

//	NewGroup 创建一个group实例
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

//	GetGroup 返回一个之前已经用NewGroup命名的group 如果没有返回nil
func GetGroup(name string) *Group {
	mu.Lock()
	g := groups[name]
	mu.Unlock()
	return g
}

func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}

	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache] hit")
		return v, nil
	}
	return g.load(key)
}

func (g *Group) load(key string) (value ByteView, err error) {
	if g.peers != nil {
		if peer, ok := g.peers.PickPeer(key); ok {
			if value, err = g.getFromPeer(peer, key); err == nil {
				return value, nil
			}
			log.Println("[GoCache] Failed to get from peer", err)
		}
	}
	return g.getLocally(key)
}

func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}

//	RegisterPeers 将实现了PeerPicker接口的HTTPPool注入到Group中
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

//	getFromPeer 使用实现了PeerGetter接口的httpGetter从访问远程节点，获取缓存值
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	bytes, err := peer.Get(g.name, key)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: bytes}, nil
}
