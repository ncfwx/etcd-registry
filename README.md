# etcd-registry

## 创建一个 Registry 实例

NewRegistry() 使用 etcd 官方的配置 clientv3.Config，需要引入一个包: github.com/coreos/etcd/clientv3

```
conf := clientv3.Config {
    Endpoints:   []string{"localhost:2379"},
    DialTimeout: 3*time.Second,
    Username:    "root",
    Password:    "root",
}
r, err := registry.NewRegistry(conf, "/example/test/")
```

## 注册当前结点

KeepRegisterNode 会起一个 goroutine 来对 etcd 的租约续期。
```
r.KeepRegisterNode(&registry.Node{IP:ip, Port:port}, 3*time.Second)
```

## 发现结点

KeepWatchNodes() 会起一个 goroutine 来保持对 etcd 的 watch。
```
r.KeepWatchNodes()
```

## 获取发现的结点

GetNodes() 是从内存中读取已发现的结点，即使 etcd 服务挂了，结点信息依然可以获得。
```
nodes := r.GetNodes()
```
