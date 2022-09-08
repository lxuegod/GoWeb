package gocache

import pb "gocache/gocache/gocachepb"

//	PeerPicker接口必须在本地实现
//	peer 拥有特定的key
type PeerPicker interface {
	PickPeer(key string) (peers PeerGetter, ok bool)
}

//	PeerGetter接口必须由peer实现
type PeerGetter interface {
	//	从对应group查找缓存值
	Get(in *pb.Request, out *pb.Response) error
}
