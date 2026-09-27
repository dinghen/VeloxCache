# VeloxCache

VeloxCache 是一个使用 Go 编写的内存分布式缓存服务。本仓库基于已有开源项目，由我主导完成二次开发和维护，重点放在单 owner 路由、缓存生命周期、并发安全和节点间通信的可靠性上。

## 项目来源

| 项目 | 地址 | 说明 |
| --- | --- | --- |
| 原项目 | [github.com/youngyangyang04/VeloxCache](https://github.com/youngyangyang04/VeloxCache) | 提供缓存组、LRU/LRU2、singleflight、gRPC 和 etcd 服务发现等基础实现 |
| 当前仓库 | [github.com/dinghen/VeloxCache](https://github.com/dinghen/VeloxCache) | 由我主导进行二次开发、问题修复和测试补充 |

项目的 Go module 路径仍保持为 `github.com/youngyangyang04/VeloxCache`，以兼容原项目的代码和依赖；仓库地址与 module 路径不同是有意保留的兼容性安排。

## 二次开发内容

本次二次开发主要完成了以下工作：

- 采用一致性哈希实现单 owner 路由：每个 key 在同一时刻只由一个节点负责。
- 统一 `Get`、`Set`、`Delete` 的路由行为：owner 节点在本地执行，非 owner 节点通过 gRPC 转发请求。
- 移除异步副本同步路径，避免重复写入、请求循环和节点间状态不明确的问题。
- 规范化节点地址，并正确识别本节点，避免 `:port`、本机 IP 和注册中心地址不一致导致的错误路由。
- 修复 singleflight 的并发竞态，确保同一个 key 只执行一次回源；回源函数 panic 时也能释放等待者并清理状态。
- 修复一致性哈希的并发访问、重平衡和后台 goroutine 生命周期问题。
- 修复 Cache、LRU 和 LRU2 在 TTL、淘汰、清空、关闭及并发访问场景下的生命周期问题。
- 修复 Group、Server 和 etcd 注册流程的停止逻辑、资源释放和配置隔离问题。
- 增加单元测试、并发测试、竞态测试和双节点 owner 路由测试，覆盖缓存及分布式请求的关键行为。

## 当前架构

```text
应用请求
   |
   v
Group                 缓存组、请求编排、owner 路由
   |
   +--> Cache          懒初始化、TTL 转换、缓存统计
   |       |
   |       +--> store  LRU / LRU2 内存存储和淘汰
   |
   +--> singleflight  合并相同 key 的并发回源
   |
   +--> PeerPicker     一致性哈希选择 owner
           |
           +--> gRPC Client/Server  节点间 Get/Set/Delete
           +--> etcd registry        节点注册与服务发现
```

主要目录和职责：

| 目录或文件 | 职责 |
| --- | --- |
| `group.go` | 缓存组、请求入口、单 owner 路由和统计 |
| `cache.go` | 缓存生命周期、懒初始化、TTL 和统计封装 |
| `store/` | LRU、LRU2 及底层存储接口 |
| `consistenthash/` | 一致性哈希环和节点重平衡 |
| `singleflight/` | 合并相同 key 的并发加载请求 |
| `client.go`、`server.go`、`peers.go` | gRPC 节点通信和 peer 选择 |
| `registry/` | etcd 注册、租约续期和注销 |
| `pb/` | protobuf 消息及 gRPC 代码 |

## 分布式语义和边界

- 每个 key 由一致性哈希环选出一个 owner。
- owner 节点直接访问自己的本地缓存和数据源。
- 非 owner 节点将请求转发给 owner，避免多个节点同时写入同一个 key。
- 节点加入或离开时，一致性哈希环会重新平衡；当前实现不会自动迁移或复制已有缓存数据。
- 当前实现是内存缓存，不提供持久化存储。
- 当前实现不提供多副本复制、故障转移或跨节点数据恢复；owner 节点不可用时，请求会返回错误。
- etcd 只负责节点注册和服务发现，不保存缓存数据。

## 已支持的能力

- Go 内存缓存
- LRU 和 LRU2 淘汰策略（默认使用 LRU2）
- TTL 过期和后台清理
- singleflight 并发回源合并
- 一致性哈希单 owner 路由
- gRPC 节点间访问
- etcd 服务注册与发现
- 缓存命中、未命中、回源和 peer 访问统计

## 快速开始

### 环境要求

- Go 1.22 或更高版本
- etcd 3.5
- Docker Compose（仅在使用仓库提供的 etcd 容器时需要）

### 启动 etcd

```bash
make etcd-up
```

也可以直接执行：

```bash
docker compose up -d etcd
```

默认 etcd 地址为 `localhost:2379`。
默认服务名为 `velox-cache`。

### 运行示例节点

在两个终端分别启动两个缓存节点：

```bash
make run-example PORT=8001 NODE=A
make run-example PORT=8002 NODE=B
```

示例程序会创建缓存组、注册节点、写入本节点数据，并演示本地 owner 和远程 owner 的读取流程。单节点运行时也可以执行：

```bash
make run-example
```

### 在代码中使用

下面是核心调用片段，其中 `loadFromSource` 需要替换为实际的数据源加载逻辑：

```go
group := veloxcache.NewGroup(
    "users",
    8<<20,
    veloxcache.GetterFunc(func(ctx context.Context, key string) ([]byte, error) {
        return loadFromSource(ctx, key)
    }),
)

value, err := group.Get(ctx, "user:1")
```

分布式部署时，为 `Group` 注册 `PeerPicker`，并使用 `Server` 和 `ClientPicker` 接入 gRPC 与 etcd 服务发现。可参考 [`example/test.go`](example/test.go)。

## 测试和质量检查

```bash
make setup
make test
make vet
```

需要检查并发竞态时执行：

```bash
go test -race ./...
```

项目的核心验证还包括重复运行测试：

```bash
go test ./... -count=3
```

停止本地 etcd：

```bash
make etcd-down
```

## 许可证

本项目使用 [MIT License](LICENSE)。
