package registry

import (
	"flag"
	"testing"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/ncfwx/x/ip"
)

var (
	registerPrefix = "/gotest/registrytest"
	localIP        string
	etcdConf       clientv3.Config
)

func init() {
	localIP, _ = ip.GetLocalIp()

	var (
		etcdEndpoints = flag.String("etcd", "your-etcd-host:2379", "your etcd hosts")
		etcdUser      = flag.String("user", "root", "your etcd user")
		etcdPassword  = flag.String("password", "root", "your etcd password")
	)

	flag.Parse()

	etcdConf = clientv3.Config{
		Endpoints:   []string{*etcdEndpoints},
		DialTimeout: 3 * time.Second,
		Username:    *etcdUser,
		Password:    *etcdPassword,
	}

}

func TestNewRegistry(t *testing.T) {
	_, err := NewRegistry(etcdConf)
	if err != nil {
		t.Errorf("new registry failed:%v", err)
		return
	}
}

func TestKeepRegisterNode(t *testing.T) {
	r, _ := NewRegistry(etcdConf)

	err := r.KeepRegisterNode(&ServiceNode{IP: localIP, Port: "60001"}, registerPrefix, 3*time.Second)
	if err != nil {
		t.Errorf("keep register node failed:%v", err)
		return
	}
}

func TestKeepWatchNodes(t *testing.T) {
	r, _ := NewRegistry(etcdConf)

	nodes := ServiceNodes{}
	r.KeepWatchNodes("/example/unittest/", &nodes)

	for i := 0; i < 3; i++ {
		if len(nodes.Nodes) > 0 {
			break
		}
		time.Sleep(1 * time.Second)
	}
	err := r.KeepRegisterNode(&ServiceNode{IP: localIP, Port: "60001"}, "/example/unittest/", 3*time.Second)
	if err != nil {
		t.Errorf("keep register node failed:%v", err)
	}

	if len(nodes.Nodes) == 0 {
		t.Errorf("keep watch node failed:%v", "watched nothing")
	}
}
