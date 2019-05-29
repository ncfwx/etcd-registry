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
	retryWait     = 3 * time.Second
	watchTimeout  = 5 * time.Minute
	requestTimout = 3 * time.Second
)

// Registry 注册器
type Registry struct {
	client *clientv3.Client
	prefix string
	nodes  []Node
}

// Node 服务结点
type Node struct {
	IP           string
	Port         string
	RegisterTime string
}

// NewRegistry 初始化一个注册实例
func NewRegistry(conf clientv3.Config, prefix string) (*Registry, error) {
	client, err := clientv3.New(conf)
	log.Printf("registry new client. conf:%+v, prefix:%v, err:%v", conf, prefix, err)
	if err != nil {
		return nil, err
	}

	return &Registry{client: client, prefix: prefix}, nil
}

// RegisterNode 注册一个结点
func (r *Registry) KeepRegisterNode(node *Node, ttl time.Duration) error {
	log.Printf("registry register node start. node:%+v, ttl:%v", node, ttl)
	leaseID, err := r.registerNode(node, ttl)

	go func(node *Node, ttl time.Duration, leaseID clientv3.LeaseID) {
		defer func() {
			err := recover()
			log.Printf("registry register node recover. err:%v", err)
			time.Sleep(retryWait)
			r.KeepRegisterNode(node, ttl)
		}()

		r.keepAlive(leaseID)
		log.Printf("registry keep alive over")
	}(node, ttl, leaseID)

	return err
}

func (r *Registry) registerNode(node *Node, ttl time.Duration) (clientv3.LeaseID, error) {
	ctx, _ := context.WithTimeout(context.Background(), requestTimout)
	leaseResponse, err := r.client.Grant(ctx, int64(ttl.Seconds()))
	log.Printf("registry grant new lease. lease:%+v, err:%v", leaseResponse, err)
	if err != nil {
		return 0, fmt.Errorf("grant lease failed:%s", err)
	}

	node.RegisterTime = time.Now().Format("2006-01-02 15:04:05")
	key := fmt.Sprintf("%s%s:%s", r.prefix, node.IP, node.Port)
	value, _ := json.Marshal(node)

	resp, err := r.client.Put(ctx, key, string(value), clientv3.WithLease(leaseResponse.ID))
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

// WatchNodes 监控注册的结点
func (r *Registry) KeepWatchNodes() {
	go func() {
		defer func() {
			err := recover()
			log.Printf("registry watch recover. err:%v", err)
			time.Sleep(retryWait)
			r.KeepWatchNodes()
		}()

		r.watchNodes()
	}()
}

// GetNodes 获取发现的结点
func (r *Registry) GetNodes() []Node {
	return r.nodes
}

func (r *Registry) watchNodes() {
	r.updateNodes()

	ctx, _ := context.WithTimeout(context.Background(), watchTimeout)
	ch := r.client.Watch(ctx, r.prefix, clientv3.WithPrefix())
	for msg := range ch {
		log.Printf("registry watched new events. msg:%+v", msg)
		r.updateNodes()
	}
	log.Printf("registry watch keep alive closed")
}

func (r *Registry) updateNodes() error {
	ctx, _ := context.WithTimeout(context.Background(), requestTimout)
	resp, err := r.client.Get(ctx, r.prefix, clientv3.WithPrefix())
	log.Printf("registry update nodes. resp:%+v, err:%v", resp, err)
	if err != nil {
		return fmt.Errorf("get value failed:%s", err)
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
