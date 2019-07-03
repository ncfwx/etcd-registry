package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

// ServiceNode 服务结点
type ServiceNode struct {
	IP           string
	Port         string
	RegisterTime string
}

// Key 返回存入etcd的Key
func (n *ServiceNode) Key() string {
	return fmt.Sprintf("%s:%s", n.IP, n.Port)
}

// Value 返回存入etcd的Value
func (n *ServiceNode) Value() (string, error) {
	n.RegisterTime = time.Now().Format("2006-01-02 15:04:05")
	data, err := json.Marshal(n)
	return string(data), err
}

// ServiceNodes 表示服务结点的集合
type ServiceNodes struct {
	Nodes []ServiceNode
}

// Put 放入一个结点
func (nodes *ServiceNodes) Put(key []byte, value []byte) error {
	node, err := parseNode(value)
	nodes.Nodes = append(nodes.Nodes, node)
	return err
}

// PutAll 放入全部结点
func (nodes *ServiceNodes) PutAll(keys [][]byte, values [][]byte) (err error) {
	result := make([]ServiceNode, len(values))
	for k, value := range values {
		result[k], err = parseNode(value)
		if err != nil {
			return fmt.Errorf("put all failed:%v", err)
		}
	}
	nodes.Nodes = result
	return nil
}

func parseNode(value []byte) (ServiceNode, error) {
	node := ServiceNode{}
	err := json.Unmarshal(value, &node)
	return node, err
}

// Delete 删除一个结点
func (nodes *ServiceNodes) Delete(key []byte) {
	fmt.Println("==== nodes Delete", key)
	for k, v := range nodes.Nodes {
		if bytes.HasSuffix(key, []byte(v.Key())) {
			nodes.Nodes = append((nodes.Nodes)[:k], (nodes.Nodes)[k+1:]...)
			return
		}
	}
}
