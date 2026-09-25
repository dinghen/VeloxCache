下一动作：用 15 分钟对照 [group.go](/home/xiao/Project/VeloxCache/group.go:42)、[cache.go](/home/xiao/Project/VeloxCache/cache.go:14) 和 `store/`，按下面三层图标出每个 `closed`、计数器和锁。当前第 2 步：理解“每一层各自保护什么”。

```text
Group
  负责业务请求和缓存组生命周期
      |
      v
Cache
  负责懒初始化、本地缓存统计、包装 Store
      |
      v
Store
  负责真正保存数据、淘汰、过期、后台清理
```

## 1. Group 层

| 字段 | 原子操作 | 责任 |
|---|---|---|
| `g.closed int32` | `LoadInt32`、`CompareAndSwapInt32` | 这个缓存组是否还接受业务请求 |
| `g.stats.localHits` | `AddInt64`、`LoadInt64` | 从 `Group.Get` 入口看，本地缓存命中次数 |
| `g.stats.localMisses` | `AddInt64`、`LoadInt64` | 从 `Group.Get` 入口看，本地缓存未命中次数 |
| `loads`、`peerHits`、`peerMisses` 等 | `AddInt64`、`LoadInt64` | 回源、远程节点、错误、耗时等业务指标 |

`Group.Get` 中：

```go
if atomic.LoadInt32(&g.closed) == 1 {
    return ByteView{}, ErrGroupClosed
}
```

意思是：这个业务组已经关闭，直接拒绝请求。

`Group.Close` 中：

```go
if !atomic.CompareAndSwapInt32(&g.closed, 0, 1) {
    return nil
}
```

意思是：

```text
如果 closed 还是 0：
    原子地改为 1
    当前 goroutine 执行真正的关闭
否则：
    说明已经关闭
    直接返回
```

所以 `Group.Close` 主要做三件事：

1. 让组进入关闭状态。
2. 调用 `g.mainCache.Close()`。
3. 从全局 `groups` 注册表中移除自己。

它保护的是“这个缓存命名空间还是否存在、还能否对外服务”。

## 2. Cache 层

[cache.go](/home/xiao/Project/VeloxCache/cache.go:14) 有四类原子字段：

| 字段 | 含义 |
|---|---|
| `closed` | `Cache` 包装器是否已经关闭 |
| `initialized` | 底层 `store.Store` 是否已经创建 |
| `hits` | `Cache.Get` 层面的命中次数 |
| `misses` | `Cache.Get` 层面的未命中次数 |

### `Cache.closed`

它保护的是 Cache 自身：

```go
if atomic.LoadInt32(&c.closed) == 1 {
    return ByteView{}, false
}
```

`Cache.Close` 会：

```text
closed: 0 -> 1
```

之后 `Add`、`Get`、`Delete`、`Clear`、`Len` 都会拒绝或返回空结果。

注意：`Cache.Close` 不会把 `closed` 改回 `0`，所以它是不可重新打开的。

### `Cache.initialized`

它保护的是底层 Store 是否存在：

```go
if atomic.LoadInt32(&c.initialized) == 0 {
    atomic.AddInt64(&c.misses, 1)
    return ByteView{}, false
}
```

这里不是“套了两层原子操作”，而是两个独立动作：

```text
Load initialized
  -> 决定是否继续查 Store

Add misses
  -> 记录这次查询失败
```

`initialized` 的生命周期是：

```text
NewCache       initialized = 0，store = nil
第一次 Add
  -> ensureInitialized 创建 LRU/LRU-2
  -> initialized = 1
Cache.Close
  -> store.Close()
  -> store = nil
  -> initialized = 0
```

但关闭后虽然 `initialized` 回到了 `0`，`closed` 仍然是 `1`，因此不会重新初始化。

### 为什么 `ensureInitialized` 要检查两次？

```go
if atomic.LoadInt32(&c.initialized) == 1 {
    return
}

c.mu.Lock()
defer c.mu.Unlock()

if c.initialized == 0 {
    c.store = store.NewStore(...)
    atomic.StoreInt32(&c.initialized, 1)
}
```

这是“双重检查”：

```text
第一次 Load：已经初始化就快速返回，避免加锁
加锁：多个 goroutine 同时初始化时，只允许一个进入
第二次检查：前一个 goroutine 可能已经完成初始化
```

这里的锁保护的是 `c.store` 这个指针以及初始化过程；原子变量只表示状态。

## 3. Store / LRU 层

底层 LRU 并没有使用 `atomic closed` 管理数据结构。

### LRU

`lruCache` 使用：

```text
sync.RWMutex
```

保护：

- `items` map
- 双向链表 `list`
- `expires`
- `usedBytes`

因为这些是多个字段必须保持一致，不能只靠原子操作。

例如添加一个元素必须同时完成：

```text
链表加入节点
items[key] = elem
usedBytes += ...
```

这类“多个对象一起变化”的操作必须用锁。

`lruCache.Close` 主要负责：

```text
停止 cleanupTicker
关闭 closeCh
结束后台 cleanupLoop
```

它不负责决定业务层是否还能调用缓存，这由 `Cache.closed` 负责。

### LRU-2

`lru2Store` 使用每个桶自己的 `sync.Mutex`，保护两个缓存层：

```text
bucket -> 一级缓存
       -> 二级缓存
```

它唯一明显使用原子的地方是内部时钟：

```go
func Now() int64 {
    return atomic.LoadInt64(&clock)
}
```

后台 goroutine 会定期：

```go
atomic.StoreInt64(&clock, time.Now().UnixNano())
atomic.AddInt64(&clock, ...)
```

这个 `clock` 不是关闭状态，也不是统计数据，而是一个共享时间戳：

```text
多个 goroutine 高频读取 clock
一个后台 goroutine 定期更新 clock
```

用原子读写可以避免每次都调用 `time.Now()`，同时保证读取到完整的 `int64` 值。

## 4. 一致性哈希层

[consistenthash/con_hash.go](/home/xiao/Project/VeloxCache/consistenthash/con_hash.go:112) 中：

```go
atomic.AddInt64(&m.totalRequests, 1)
```

它记录节点路由请求总数，用于判断是否需要重新平衡虚拟节点。

相关流程：

```text
Map.Get
  -> 选择节点
  -> totalRequests + 1

后台 balancer
  -> Load totalRequests
  -> 达到 1000 次后检查负载
  -> 重新平衡后 Store(totalRequests, 0)
```

这里原子保护的只是 `totalRequests` 这个整数，不会自动保护：

```go
m.nodeCounts
m.keys
m.hashMap
```

这些仍然需要 `m.mu`。

阅读时可以特别注意：当前代码中 `nodeCounts` 在 `RLock` 下被修改，以及部分地方直接读取 `m.totalRequests`。这说明“用了 atomic”不等于“整段并发逻辑都安全”，原子操作只能保护它直接操作的那个变量。

## 最重要的区别

```text
Group.Close
  关闭业务组
  拒绝 Group.Get / Set / Delete
  从全局组注册表移除

Cache.Close
  关闭本地缓存包装器
  关闭底层 Store
  停止继续访问或初始化 Store

Store.Close
  释放底层缓存实现的资源
  停止 LRU/LRU-2 的后台清理任务
```

而统计字段的关系是：

```text
Cache.misses
  = Cache.Get 这一层查找失败的次数

Group.stats.localMisses
  = Group.Get 这一层本地缓存未命中的次数
```

一次 `Group.Get` 的本地未命中，通常会同时增加这两个计数，因为它经过了两层。

下一步只做一个小练习：分别读 `Group.Close` 和 `Cache.Close`，回答一句话：

> 为什么 `Group.Close` 不能只设置 `g.closed = 1`，还必须调用 `g.mainCache.Close()`？