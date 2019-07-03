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

// Node 存入etcd的结点
type Node interface {
	GetKey() string
	GetValue() (string, error)
}

// Nodes 一个prefix下的结点集合
type Nodes interface {
	Put(key []byte, data []byte) error
	Delete(key []byte)
	Clear()
}

// NewRegistry 初始化一个注册实例
func NewRegistry(conf clientv3.Config) (*Registry, error) {
	client, err := clientv3.New(conf)
	log.Printf("registry new client. conf:%+v, err:%v", conf, err)
	if err != nil {
		return nil, err
	}

	return &Registry{client: client}, nil
}

// KeepRegisterNode 注册一个结点
func (r *Registry) KeepRegisterNode(node Node, prefix string, ttl time.Duration) error {
	log.Printf("registry register node start. node:%+v, prefix:%v, ttl:%v", node, prefix, ttl)

	leaseID, err := r.registerNode(node, prefix, ttl)

	go func(node Node, prefix string, ttl time.Duration, leaseID clientv3.LeaseID) {
		defer func() {
			err := recover()
			log.Printf("registry register node recover. err:%v", err)
			time.Sleep(retryWait)
			r.KeepRegisterNode(node, prefix, ttl)
		}()

		r.keepAlive(leaseID)
		log.Printf("registry keep alive over")
	}(node, prefix, ttl, leaseID)

	return err
}

func (r *Registry) registerNode(node Node, prefix string, ttl time.Duration) (clientv3.LeaseID, error) {
	ctx, _ := context.WithTimeout(context.Background(), requestTimout)

	leaseResponse, err := r.client.Grant(ctx, int64(ttl.Seconds()))
	log.Printf("registry grant new lease. lease:%+v, err:%v", leaseResponse, err)
	if err != nil {
		return 0, fmt.Errorf("grant lease failed:%s", err)
	}

	key := prefix + node.GetKey()
	value, _ := node.GetValue()

	resp, err := r.client.Put(ctx, key, value, clientv3.WithLease(leaseResponse.ID))
	log.Printf("registry put. resp:%+v, key:%s, value:%s, err:%v", resp, key, value, err)
	if err != nil {
		return 0, fmt.Errorf("put value failed:%s", err)
	}

	return leaseResponse.ID, nil
}

func (r *Registry) keepAlive(leaseID clientv3.LeaseID) {
	ch, err := r.client.KeepAlive(context.Background(), leaseID)
	log.Printf("registry keep alive start. err:%v", err)
	if err != nil {
		return
	}

	for range ch {
		// LeaseKeepAliveResponse
	}
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

func (r *Registry) updateNodes(prefix string, nodes Nodes) error {
	ctx, _ := context.WithTimeout(context.Background(), requestTimout)
	resp, err := r.client.Get(ctx, prefix, clientv3.WithPrefix())

	log.Printf("registry update nodes. resp:%+v, err:%v", resp, err)
	if err != nil {
		return fmt.Errorf("get value failed:%s", err)
	}

	nodes.Clear()
	for _, v := range resp.Kvs {
		nodes.Put(v.Key, v.Value)
	}

	return nil
}
