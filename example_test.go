package registry_test

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/ncfwx/etcd-registry/registry"
	"github.com/ncfwx/x/ip"
)

func Example() {
	conf := clientv3.Config{
		Endpoints:   []string{"your-etcd-host:2379"},
		DialTimeout: 3 * time.Second,
		Username:    "root",
		Password:    "devdev",
	}

	r, err := registry.NewRegistry(conf)
	if err != nil {
		log.Printf("[example] new registry failed. err:%v", err)
		return
	}

	prefix := "/example/test/"
	ip, _ := ip.GetLocalIp()
	rand.Seed(time.Now().UnixNano())
	port := fmt.Sprintf("%v", rand.Intn(10000)+10000)

	// register
	r.KeepRegisterNode(&registry.ServiceNode{IP: ip, Port: port}, prefix, 3*time.Second)

	// discovery
	nodes := registry.ServiceNodes{}
	r.KeepWatchNodes("/example/test/", &nodes)

	for i := 0; i < 100; i++ {
		log.Printf("[example] nodes = %v", len(nodes))
		for k, v := range nodes {
			log.Printf("[example] k = %v, v = %v", k, v)
		}
		time.Sleep(3 * time.Second)
	}

	fmt.Printf("done")
	// Output: done
}
