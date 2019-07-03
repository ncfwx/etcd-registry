package registry

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/coreos/etcd/clientv3"
)

const (
	retryWait     = 3 * time.Second
	requestTimout = 3 * time.Second
	watchTimeout  = 10 * time.Minute
)

// Registry 注册器
type Registry struct {
	client *clientv3.Client
}

// Node 表示一个存入etcd的结点数据
type Node interface {
	Key() string
	Value() (string, error)
}

// Nodes 表示一个prefix下的所有数据的集合
type Nodes interface {
	Put(key []byte, value []byte) error
	PutAll(keys [][]byte, values [][]byte) error
	Delete(key []byte)
}

// NewRegistry 初始化一个注册实例
func NewRegistry(conf clientv3.Config) (*Registry, error) {
	client, err := clientv3.New(conf)
	if err != nil {
		return nil, fmt.Errorf("new clientv3 failed:%v", err)
	}

	return &Registry{client: client}, nil
}

// KeepRegisterNode 注册一个结点
func (r *Registry) KeepRegisterNode(node Node, prefix string, ttl time.Duration) error {
	leaseID, err := r.registerNode(node, prefix, ttl)
	log.Printf("registry register node. err:%v, node:%+v, prefix:%v, ttl:%v", err, node, prefix, ttl)

	go func(node Node, prefix string, ttl time.Duration, leaseID clientv3.LeaseID) {
		defer func() {
			err := recover()
			log.Printf("registry register node recover. err:%v", err)
			time.Sleep(retryWait)

			r.KeepRegisterNode(node, prefix, ttl)
		}()

		err := r.keepAlive(leaseID)
		log.Printf("registry keep alive finished. err:%v", err)
	}(node, prefix, ttl, leaseID)

	return nil
}

func (r *Registry) registerNode(node Node, prefix string, ttl time.Duration) (clientv3.LeaseID, error) {
	ctx, _ := context.WithTimeout(context.Background(), requestTimout)
	leaseResponse, err := r.client.Grant(ctx, int64(ttl.Seconds()))

	log.Printf("registry grant new lease. err:%v, lease:%+v", err, leaseResponse)
	if err != nil {
		return 0, fmt.Errorf("grant lease failed:%v", err)
	}

	key := prefix + node.Key()
	value, _ := node.Value()
	resp, err := r.client.Put(ctx, key, value, clientv3.WithLease(leaseResponse.ID))

	log.Printf("registry put value.  err:%v, resp:%+v, key:%s, value:%s", err, resp, key, value)
	if err != nil {
		return 0, fmt.Errorf("put value failed:%v", err)
	}

	return leaseResponse.ID, nil
}

func (r *Registry) keepAlive(leaseID clientv3.LeaseID) error {
	ch, err := r.client.KeepAlive(context.Background(), leaseID)

	log.Printf("registry keep alive start. err:%v", err)
	if err != nil {
		return fmt.Errorf("keepAlive failed:%v", err)
	}

	for range ch {
		// nothing to do
	}

	return fmt.Errorf("keepAlive channel closed")
}

// KeepWatchNodes 监控注册的结点
func (r *Registry) KeepWatchNodes(prefix string, nodes Nodes) {
	go func(prefix string, nodes Nodes) {
		defer func() {
			err := recover()
			log.Printf("registry watch recover. err:%v", err)
			time.Sleep(retryWait)

			r.KeepWatchNodes(prefix, nodes)
		}()

		r.watchNodes(prefix, nodes)
	}(prefix, nodes)
}

func (r *Registry) watchNodes(prefix string, nodes Nodes) {
	r.updateNodes(prefix, nodes)

	ctx, _ := context.WithTimeout(context.Background(), watchTimeout)
	ch := r.client.Watch(ctx, prefix, clientv3.WithPrefix())

	for msg := range ch {
		log.Printf("registry watched new events. msg:%+v", msg)
		for _, event := range msg.Events {
			r.handleEvent(event, nodes)
		}
	}

	log.Printf("registry watch keep alive closed")
}

func (r *Registry) updateNodes(prefix string, nodes Nodes) {
	ctx, _ := context.WithTimeout(context.Background(), requestTimout)
	resp, err := r.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		log.Printf("registry update nodes get failed. err:%v", err)
		return
	}
	log.Printf("registry update nodes get success. resp:%+v", resp)

	keys := [][]byte{}
	values := [][]byte{}
	for _, v := range resp.Kvs {
		keys = append(keys, v.Key)
		values = append(values, v.Value)
	}

	err = nodes.PutAll(keys, values)
	if err != nil {
		log.Printf("registry update nodes put failed. err:%v", err)
	}
}

func (r *Registry) handleEvent(event *clientv3.Event, nodes Nodes) {
	switch {
	case event.IsCreate():
		nodes.Put(event.Kv.Key, event.Kv.Value)
	case event.IsModify():
		nodes.Delete(event.Kv.Key)
		nodes.Put(event.Kv.Key, event.Kv.Value)
	case event.Type == clientv3.EventTypeDelete:
		nodes.Delete(event.Kv.Key)
	}
}
