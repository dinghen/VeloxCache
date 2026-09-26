package registry

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// Config 定义etcd客户端配置
type Config struct {
	Endpoints   []string      // 集群地址
	DialTimeout time.Duration // 连接超时时间
}

// DefaultConfig 提供默认配置
var DefaultConfig = &Config{
	Endpoints:   []string{"localhost:2379"},
	DialTimeout: 5 * time.Second,
}

// Register 注册服务到etcd
func Register(svcName, addr string, stopCh <-chan error) error {
	return RegisterWithConfig(svcName, addr, stopCh, *DefaultConfig)
}

// RegisterWithConfig registers a service using an instance-specific etcd configuration.
func RegisterWithConfig(svcName, addr string, stopCh <-chan error, config Config) error {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   append([]string(nil), config.Endpoints...),
		DialTimeout: config.DialTimeout,
	})
	if err != nil {
		return fmt.Errorf("failed to create etcd client: %v", err)
	}

	addr, err = NormalizeAddr(addr)
	if err != nil {
		cli.Close()
		return fmt.Errorf("failed to normalize service address: %w", err)
	}

	// 创建租约
	lease, err := cli.Grant(context.Background(), 10) // 增加租约时间到10秒
	if err != nil {
		cli.Close()
		return fmt.Errorf("failed to create lease: %v", err)
	}

	// 注册服务，使用完整的key路径
	key := fmt.Sprintf("/services/%s/%s", svcName, addr)
	_, err = cli.Put(context.Background(), key, addr, clientv3.WithLease(lease.ID))
	if err != nil {
		cli.Close()
		return fmt.Errorf("failed to put key-value to etcd: %v", err)
	}

	// 保持租约
	keepAliveCh, err := cli.KeepAlive(context.Background(), lease.ID)
	if err != nil {
		cli.Close()
		return fmt.Errorf("failed to keep lease alive: %v", err)
	}

	// 处理租约续期和服务注销
	go func() {
		defer cli.Close()
		for {
			select {
			case <-stopCh:
				// 服务注销，撤销租约
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				cli.Revoke(ctx, lease.ID)
				cancel()
				return
			case resp, ok := <-keepAliveCh:
				if !ok {
					logrus.Warn("keep alive channel closed")
					return
				}
				logrus.Debugf("successfully renewed lease: %d", resp.ID)
			}
		}
	}()

	logrus.Infof("Service registered: %s at %s", svcName, addr)
	return nil
}

// NormalizeAddr returns the address form used in the registry and hash ring.
func NormalizeAddr(addr string) (string, error) {
	if addr == "" {
		return "", fmt.Errorf("empty address")
	}
	if addr[0] != ':' {
		return addr, nil
	}
	ip, err := getLocalIP()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%s", ip, addr), nil
}

func getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String(), nil
			}
		}
	}

	return "", fmt.Errorf("no valid local IP found")
}
