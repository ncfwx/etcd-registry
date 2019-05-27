package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/coreos/etcd/clientv3"
)

const (
	retryWait    = 3 * time.Second
	watchTimeout = 5 * time.Minute
)

// Registry 注册器
type Registry struct {
	client  *clientv3.Client
	prefix  string
	timeout time.Duration
	nodes   []Node
}

// Node 服务结点
type Node struct {
	IP           string
	Port         string
	RegisterTime string
}

// NewRegistry 初始化一个注册实例
func NewRegistry(endpoints []string, prefix string, timeout time.Duration) (*Registry, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: timeout,
	})
	log.Printf("registry new client. endpoints:%v, prefix:%v, timeout:%v, err:%v", endpoints, prefix, timeout, err)
	if err != nil {
		return nil, err
	}

	return &Registry{client: client, prefix: prefix, timeout: timeout}, nil
}

// RegisterNode 注册一个结点
func (r *Registry) RegisterNode(node *Node, ttl time.Duration) {
	go func(node *Node, ttl time.Duration) {
		defer func() {
			err := recover()
			log.Printf("registry register node recover. err:%v", err)
			time.Sleep(retryWait)
			r.RegisterNode(node, ttl)
		}()

		r.registerNode(node, ttl)
	}(node, ttl)
}

func (r *Registry) registerNode(node *Node, ttl time.Duration) {
	log.Printf("registry register node start. node:%+v, ttl:%v", node, ttl)

	ctx, _ := context.WithTimeout(context.Background(), r.timeout)
	lease, err := r.client.Grant(ctx, int64(ttl.Seconds()))
	log.Printf("registry grant new lease. lease:%+v, err:%v", lease, err)
	if err != nil {
		return
	}

	node.RegisterTime = time.Now().Format("2006-01-02 15:04:05")
	key := fmt.Sprintf("%s%s:%s", r.prefix, node.IP, node.Port)
	value, _ := json.Marshal(node)
	resp, err := r.client.Put(ctx, key, string(value), clientv3.WithLease(lease.ID))
	log.Printf("registry put. resp:%+v, key:%s, value:%s, err:%v", resp, key, value, err)
	if err != nil {
		return
	}

	ch, err := r.client.KeepAlive(context.TODO(), lease.ID)
	log.Printf("registry keep alive start. err:%v", err)
	if err != nil {
		return
	}

	for {
		_, ok := <-ch
		if !ok {
			log.Printf("registry keep alive closed.")
			return
		}
	}
}

// WatchNodes 监控注册的结点
func (r *Registry) WatchNodes() {
	go func() {
		defer func() {
			err := recover()
			log.Printf("registry watch recover. err:%v", err)
			time.Sleep(retryWait)
			r.WatchNodes()
		}()

		r.watchNodes()
	}()
}

// GetNodes 获取注册的结点
func (r *Registry) GetNodes() []Node {
	return r.nodes
}

func (r *Registry) watchNodes() {
	r.updateNodes()
	ctx, _ := context.WithTimeout(context.Background(), watchTimeout)
	ch := r.client.Watch(ctx, r.prefix, clientv3.WithPrefix())
	for msg := range ch {
		log.Printf("registry watch new events. msg:%+v", msg)
		r.updateNodes()
	}
	log.Printf("registry watch keep alive closed.")
}

func (r *Registry) updateNodes() error {
	ctx, _ := context.WithTimeout(context.Background(), r.timeout)
	resp, err := r.client.Get(ctx, r.prefix, clientv3.WithPrefix())
	log.Printf("registry update nodes. resp:%+v, err:%v", resp, err)
	if err != nil {
		return err
	}

	nodes := []Node{}
	for _, v := range resp.Kvs {
		node := Node{}
		json.Unmarshal(v.Value, &node)
		nodes = append(nodes, node)
	}

	r.nodes = nodes
	return nil
}
