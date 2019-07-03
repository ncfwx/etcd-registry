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

// GetKey 获取存入etcd的Key
func (n *ServiceNode) GetKey() string {
	return fmt.Sprintf("%s:%s", n.IP, n.Port)
}

// GetValue 获取存入etcd的Value
func (n *ServiceNode) GetValue() (string, error) {
	n.RegisterTime = time.Now().Format("2006-01-02 15:04:05")
	data, err := json.Marshal(n)
	return string(data), err
}

// ServiceNodes 服务结点集合
type ServiceNodes []ServiceNode

// Put 放入一个结点
func (nodes *ServiceNodes) Put(key []byte, value []byte) error {
	nodes.Delete(key)

	node := ServiceNode{}
	err := json.Unmarshal(value, &node)
	*nodes = append(*nodes, node)

	return err
}

// Delete 删除一个结点
func (nodes *ServiceNodes) Delete(key []byte) {
	for k, v := range *nodes {
		if bytes.HasSuffix(key, []byte(v.GetKey())) {
			*nodes = append((*nodes)[:k], (*nodes)[k+1:]...)
			return
		}
	}
}

// Clear 清空所有结点
func (nodes *ServiceNodes) Clear() {
	*nodes = (*nodes)[0:0]
}
