package registry_test

import (
	"fmt"
	"log"
	"time"
	"math/rand"

	"github.com/ncfwx/x/ip"
	"github.com/coreos/etcd/clientv3"
	"github.com/ncfwx/etcd-registry/registry"
)

func Example() {
	conf := clientv3.Config {
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 3*time.Second,
		Username:    "root",
		Password:    "devdev",
	}

	r, err := registry.NewRegistry(conf, "/example/test/")
	if err != nil {
		log.Printf("[example] new registry failed. err:%v", err)
		return
	}

	// register
	ip, _ := ip. GetLocalIp()
	rand.Seed(time.Now().UnixNano())
	port := fmt.Sprintf("%v", rand.Intn(10000) + 10000)
	r.KeepRegisterNode(&registry.Node{IP:ip, Port:port}, 3*time.Second)

	// discovery
	r.KeepWatchNodes()
	log.Printf("[example] r = %+v", r)

	for i := 0; i < 100; i++ {
		nodes := r.GetNodes()
		log.Printf("[example] nodes = %+v", nodes)
		time.Sleep(3 * time.Second)
	}

	fmt.Printf("done")
	// Output: done
}
