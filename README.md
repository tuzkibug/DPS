# DPS — DNS Packet Sender

高性能 DNS 请求发包机，支持 CSV 域名列表和 PCAP 抓包文件两种输入模式，通过 raw socket 发送 DNS 查询包，可灵活控制发包速率、抖动和延迟。

## 功能特性

- **双输入模式**：CSV 域名列表（UDP socket）/ PCAP 抓包文件回放（AF_PACKET raw socket，以太网帧级地址改写）
- **随机源地址**：支持每包随机源 IP 和随机源 MAC，模拟海量客户端 DNS 请求；CSV 模式自动切换为 AF_PACKET raw socket 构建完整帧
- **网口指定**：可显式指定发包网口名称（如 eth0），替代原有基于源 IP 的自动反查
- **PCAP 服务器路径**：无需上传，直接选择服务器端 PCAP 目录或文件，支持按日期组织
- **QoS 控制**：精细控制发包速率（QPS）、抖动（Jitter）、延迟（Delay），高 QPS 自动批处理
- **实时监控**：WebSocket 每秒推送统计（当前 QPS、失败数、运行时长、累计运行时长）
- **任务编辑**：非运行状态下可修改任务配置（名称、路径、IP/MAC、QoS 参数）
- **Web 控制台**：React 18 + Ant Design 5 现代前端界面，PCAP 目录浏览器
- **可配置端口**：通过 `.env` 文件自定义所有端口，适配任意部署环境
- **重启恢复**：后端重启后自动恢复之前运行中的任务
- **WebSocket 来源校验**：仅允许配置的前端域名连接，防止跨站 WebSocket 劫持

## 项目架构

```
┌─────────────────────────────────────────────────────────────────┐
│                     Frontend (React 18 + TypeScript)              │
│  Pages: TaskList / TaskCreate / TaskDetail                       │
│  Components: TaskForm / TaskList / LiveMonitor / EditTaskModal   │
│  State: Zustand                                                  │
│  UI: Ant Design 5                                                │
│  Tests: vitest + @testing-library/react                          │
└──────────────────────────────┬──────────────────────────────────┘
                               │ HTTP REST + WebSocket
                               │ (nginx reverse proxy → Gin)
┌──────────────────────────────▼──────────────────────────────────┐
│                     Backend (Go 1.22 + Gin)                       │
│                                                                   │
│  ┌─────────────┐   ┌──────────────┐   ┌─────────────────────┐   │
│  │  api/handler │   │  scheduler   │   │  engine             │   │
│  │  task.go     │   │  scheduler   │   │  dns.go   (packet)  │   │
│  │  websocket   │──▶│  .go         │──▶│  pcap.go  (replay)  │   │
│  │  .go         │   │              │   │  sender.go(socket)  │   │
│  └─────────────┘   └──────┬───────┘   └─────────┬───────────┘   │
│                            │                     │                │
│                     ┌──────▼──────┐      ┌──────▼────────┐      │
│                     │    store    │      │  Raw Socket    │      │
│                     │ SQLite+Redis│      │  (UDP 53)      │      │
│                     └─────────────┘      └───────────────┘      │
└─────────────────────────────────────────────────────────────────┘
```

### 技术栈

| 层级 | 技术 |
|------|------|
| **前端** | React 18, TypeScript, Vite 5, Ant Design 5, Zustand |
| **后端** | Go 1.22, Gin, gorilla/websocket, google/gopacket, google/uuid |
| **存储** | SQLite (任务持久化), Redis (实时状态与统计) |
| **发包** | UDP socket (`net.ListenPacket`) / AF_PACKET raw socket |
| **实时推送** | WebSocket (gorilla/websocket) |
| **测试** | Go `testing` (57 测试), vitest + @testing-library/react (11 测试) |
| **部署** | Docker + Docker Compose, host 网络模式 |

### 目录结构

```
DPS/
├── backend/
│   ├── cmd/server/main.go            # 程序入口，路由注册
│   └── internal/
│       ├── api/handler/
│       │   ├── task.go               # REST API (CRUD + start/stop/stats + file download)
│       │   └── websocket.go          # WebSocket 实时推送 (含 Origin 校验)
│       ├── engine/
│       │   ├── dns.go                # DNS 包构造、域名解析/校验、QoS 控制器
│       │   ├── dns_test.go
│       │   ├── pcap.go               # PCAP 解析 (gopacket)、包地址改写、raw socket
│       │   ├── pcap_test.go
│       │   ├── sender.go             # UDP 发包器 + PCAP 回放器
│       │   └── sender_test.go
│       ├── scheduler/
│       │   ├── scheduler.go          # 任务调度、生命周期管理、重启恢复
│       │   └── scheduler_test.go
│       └── store/
│           ├── sqlite.go             # SQLite 持久化 + schema 迁移
│           ├── sqlite_test.go
│           └── redis.go              # Redis 实时状态
├── web-ui/
│   ├── nginx.conf                    # 生产 Nginx 配置（含 API 代理 + WS 升级）
│   ├── vite.config.ts                # Vite 配置（开发代理）
│   ├── vitest.config.ts              # Vitest 测试配置
│   └── src/
│       ├── App.tsx                    # 路由配置
│       ├── main.tsx                   # 入口
│       ├── test-setup.ts             # 测试初始化（jsdom polyfill）
│       ├── api/
│       │   ├── index.ts              # Axios 实例 + API 方法
│       │   └── types.ts              # TypeScript 类型定义
│       ├── components/
│       │   ├── TaskForm.tsx           # 任务创建表单
│       │   ├── TaskForm.test.tsx
│       │   ├── TaskList.tsx           # 任务列表表格
│       │   ├── EditTaskModal.tsx      # 编辑任务弹窗
│       │   └── LiveMonitor.tsx        # 实时监控面板
│       ├── pages/
│       │   ├── TaskListPage.tsx       # 任务列表页
│       │   ├── TaskCreatePage.tsx     # 创建任务页
│       │   └── TaskDetailPage.tsx     # 任务详情页
│       └── stores/
│           ├── taskStore.ts           # Zustand 状态管理
│           └── taskStore.test.ts
├── docker-compose.yml                # 三服务编排 (Redis + Backend + Frontend)
└── domain_list_example.csv           # CSV 示例文件
```

## 快速开始

### 环境要求

- Docker & Docker Compose
- （本地开发）Go 1.22+, Node.js 18+

### Docker 部署

```bash
cd DPS

# 启动全部服务（Redis + Backend + Frontend）
docker compose up -d

# 查看运行状态
docker compose ps
```

服务地址：

| 服务 | 地址 |
|------|------|
| 前端界面 | http://localhost:3000 |
| 后端 API | http://localhost:8080 |

> Backend 和 Frontend 使用 `network_mode: host` 共享宿主机网络栈，确保 raw socket (AF_PACKET) 可直接绑定物理网卡发送 PCAP 回放流量。

#### 自定义端口

若默认端口被占用，创建 `.env` 文件（参考 `.env.example`）：

```
REDIS_PORT=6380
BACKEND_PORT=9090
FRONTEND_PORT=4000
```

`docker compose up -d` 会自动读取。

### 本地开发

**后端：**

```bash
cd backend

# 确保 Redis 已运行（可使用 Docker 单独启动）
docker run -d --name redis -p 6379:6379 redis:7-alpine

# 启动后端
export REDIS_ADDR=localhost:6379
export SQLITE_PATH=./data.db
export UPLOAD_DIR=./uploads
go run cmd/server/main.go
```

**前端：**

```bash
cd web-ui
npm install
npm run dev     # Vite 开发服务器，自动代理 /api → localhost:8080
```

## 使用指南

### 1. 准备数据

**CSV 模式**：创建域名列表文件，每行一个域名：

```
www.example.com
www.google.com
dns.example.org
```

**PCAP 模式**：将 PCAP 文件放入 `./pcap/` 目录（Docker 自动挂载），可按日期组织子目录：

```
pcap/
├── 2025/10/24/dns_traffic.pcap
├── test.pcap
```

### 2. 创建任务

访问 http://localhost:3000 → 点击 **Create Task**，填写配置：

| 字段 | 说明 | 示例 |
|------|------|------|
| Task Name | 任务名称 | `DNS压力测试` |
| Input Type | CSV（域名列表）/ PCAP（包回放） | `csv` |
| File | CSV：上传文件；PCAP：浏览服务器端目录或输入路径 | `test.pcap` |
| Source IP | 源 IP 地址 | `10.0.2.15` |
| Destination IP | 目标 DNS 服务器 IP | `8.8.8.8` |
| Source MAC | 源 MAC 地址 | `08:00:27:ad:db:96` |
| Destination MAC | 目标 MAC 地址 / 网关 MAC | `52:55:0a:00:02:02` |
| Random Source IP | 每包随机生成源 IP（开启后需指定 Network Interface） | `否` |
| Random Source MAC | 每包随机生成源 MAC（开启 Random Source IP 后可见） | `否` |
| Network Interface | 发包网口名称（开启随机源 IP 时必填） | `eth0` |
| Target QPS | 目标每秒发包数 | `100` |
| Jitter | 速率抖动比例 (0-1) | `0` |
| Min/Max Delay | 每包额外延迟范围 (ms) | `0` / `0` |

> PCAP 模式会将包内原有的源/目的 MAC 和 IP 全部替换为配置值，重算校验和后通过 raw socket 发送。CSV 模式开启随机源 IP/MAC 后同样切换为 AF_PACKET raw socket，手动构建完整的 L2-L7 包并注入随机地址。

### 3. 操作任务

- **查看列表**：Tasks 页面展示所有任务，含名称、类型、目标 IP、状态
- **点击任务名**：进入详情页，查看完整配置；CSV 任务可下载上传文件
- **Edit**：非运行状态下可编辑任务配置（名称、路径、IP/MAC、QoS）
- **Start**：启动发包，状态变为 `running`
- **Stop**：停止发包，状态恢复 `pending`
- **Delete**：删除任务及关联的上传文件

### 4. 实时监控

任务启动后，实时面板展示：

- **Current QPS**：当前实际每秒发包数
- **Failed**：发包失败数
- **Current Run Time**：本次运行时长
- **Created**：任务创建日期时间
- **Last Run**：最后一次启动日期时间
- **Total Run Time**：累计运行总时长

监控数据通过 WebSocket 每秒推送，刷新页面后状态保持。

## API 参考

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/v1/tasks` | 创建任务 |
| `GET` | `/api/v1/tasks` | 获取任务列表 |
| `GET` | `/api/v1/tasks/:id` | 获取任务详情 |
| `PUT` | `/api/v1/tasks/:id` | 更新任务配置 |
| `DELETE` | `/api/v1/tasks/:id` | 删除任务（含上传文件） |
| `POST` | `/api/v1/tasks/:id/start` | 启动发包 |
| `POST` | `/api/v1/tasks/:id/stop` | 停止发包 |
| `GET` | `/api/v1/tasks/:id/stats` | 获取实时统计 |
| `GET` | `/api/v1/tasks/:id/status` | 获取任务状态 |
| `GET` | `/api/v1/tasks/:id/file` | 下载上传文件 |
| `GET` | `/api/v1/pcap/dirs` | 浏览 PCAP 目录 |
| `WS` | `/api/v1/ws/tasks/:id` | WebSocket 实时推送 |

### curl 示例

```bash
# 创建任务
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "DNS Benchmark",
    "input_type": "csv",
    "src_ip": "10.0.2.15",
    "dst_ip": "8.8.8.8",
    "src_mac": "aa:bb:cc:dd:ee:ff",
    "dst_mac": "11:22:33:44:55:66",
    "interface": "eth0",
    "random_src_ip": true,
    "random_src_mac": false,
    "qos": {
      "target_qps": 500,
      "jitter": 0.05,
      "delay_min_ms": 0,
      "delay_max_ms": 0
    }
  }'

# 启动任务
curl -X POST http://localhost:8080/api/v1/tasks/<task-id>/start

# 查看统计
curl http://localhost:8080/api/v1/tasks/<task-id>/stats
```

## QoS 参数详解

| 参数 | 含义 | 工作机制 |
|------|------|----------|
| `target_qps` | 目标每秒发包数 | QPS ≤ 2000 时每包间隔 = 1s/QPS；> 2000 QPS 自动批处理 |
| `jitter` | 速率抖动 (0-1) | 在发包间隔上叠加 ±jitter% 的随机偏差，模拟真实流量波动 |
| `delay_min_ms` | 最小额外延迟 | 与 delay_max 配合生成随机延迟 |
| `delay_max_ms` | 最大额外延迟 | 额外延迟 = random[min, max]，模拟网络延迟场景 |

## 运行测试

**后端：**

```bash
cd backend
go test ./... -v
```

**前端：**

```bash
cd web-ui
npm test           # 单次运行
npm run test:watch # 监视模式
```

## 注意事项

1. Backend 使用 host 网络模式 + `NET_ADMIN` + `NET_RAW`，确保 raw socket 可绑定物理网卡
2. CSV 模式使用标准 UDP socket，内核处理路由；PCAP 模式使用 AF_PACKET raw socket，直接发送以太网帧
3. 后端重启后，之前运行中的任务会被自动恢复；若 PCAP 文件缺失则恢复失败并自动置为 pending
4. 确保目标 DNS 服务器的 UDP 53 端口可达
5. 高 QPS 发包会消耗大量 CPU 和网络带宽，请酌情使用
6. PCAP 模式仅支持 Ethernet + IPv4 帧格式

## 版本历史

### v0.3.1 (2026-05-15) — 随机源地址与显式网口绑定

- **新增** 随机源 IP 支持：每包随机生成 IPv4 源地址，CSV 模式自动切换为 AF_PACKET raw socket 构建完整以太网帧
- **新增** 随机源 MAC 支持：每包随机生成单播、本地管理的 MAC 地址
- **新增** 显式网口名称配置（`interface` 字段），替代原有基于源 IP 的网口反查
- **新增** 前端级联 UI：Random Source IP 开关 → 展开 Random Source MAC + Network Interface 字段
- **新增** 创建任务时校验：开启随机源 IP 时必须指定网口名称
- **重构** `openRawSocket` 提取 `bindRawSocket` / `openRawSocketByName`，支持按名称绑定网口
- **新增** `loadRawPacketsFromPath` / `readRawFile`，PCAP 随机模式按需逐包改写地址

### v0.2.7 (2026-05-13) — 测试基础设施与代码去重

- **重构** 提取共享 `testutil.MockRedis`，消除 handler 和 scheduler 测试的 mock 重复代码
- **清理** 移除 `sqlite.go` 中 `GetTask`/`ListTasks` 的冗余 `uuid.Parse`
- **统一** `math/rand` → `math/rand/v2`，dns.go 与 sender.go 保持一致
- **新增** Makefile（`make test`、`build`、`vet`、`run` 等）
- **新增** TaskListPage 加载 loading 状态
- **新增** `getInterfaceByIP` 接口缓存（减少网卡枚举）
- **新增** `docker-compose.yml` 添加 `version: "3.8"`
- **清理** 删除 `uploads/` 残留开发文件

### v0.2.6 (2026-05-13) — 代码质量与可靠性

- **修复** PCAP 类型断言检查返回值，防止 nil pointer panic
- **修复** 上传文件权限 `0644` → `0600`
- **新增** QoS 参数输入校验（jitter∈[0,1]、delay≥0、delay_max≥delay_min）
- **新增** PCAP 文件/包跳过时输出日志
- **新增** TaskList 手动刷新按钮
- **新增** LiveMonitor WebSocket `onerror`/`onclose` 处理
- **改进** TaskForm `handleSubmit` 和 `breadcrumbItems` 使用 React Hooks 优化
- **修复** TaskDetailPage 区分 loading 与 not-found 状态
- **修复** `deleteTask` 添加 try/catch 错误处理
- **新增** WebSocket ping/pong 保活 + 60s 读写超时
- **改进** 后端 `PORT` → `BACKEND_PORT` 统一命名
- **新增** Backend Docker 健康检查
- **改进** 前端 Dockerfile 使用 `npm ci`

### v0.2.5 (2026-05-13) — 安全加固与可观测性

- **新增** WebSocket 关键路径错误日志（upgrade 失败、push 失败）
- **修复** `sendPacket` 失败时不再错误递增 `sentCount`
- **改进** TaskForm `handleSubmit` 用 `TaskFormValues` 接口替代 `any` 类型
- **修复** WebSocket URL 根据页面协议动态选择 `ws://` 或 `wss://`
- **新增** Backend 和 Web-UI 的 `.dockerignore` 文件
- **新增** Nginx 安全头（X-Content-Type-Options、X-Frame-Options、X-XSS-Protection、Referrer-Policy、CSP）

### v0.2.4 (2026-05-12) — 错误处理增强与集成测试

- **修复** `sqlite.go` 所有 `rows.Next()` 循环后添加 `rows.Err()` 检查
- **修复** `recoverTasks`、`StopTask`、`watchStats` 中的错误静默丢弃，改为 `log.Printf` 记录
- **修复** `StopTask` 中 `GetStartTime` 错误导致 `TotalRunMs` 从不累加的 bug
- **修复** `go.mod` 中 `gopacket` 标记为 `// indirect`
- **修复** `go vet` 报错：删除未使用的 `burstSize` 字段
- **新增** 9 个端到端集成测试（API → scheduler → engine → Redis 全链路）
- **移除** 未使用的 `recharts` 前端依赖

### v0.2.3 (2026-05-12) — 健壮性与安全加固

- **修复** `BuildUDPPacket` 包级全局变量 `srcIP`/`dstIP` 的并发数据竞争
- **新增** 后端重启自动恢复运行中的任务（扫描 SQLite status=running 并重建 sender）
- **新增** WebSocket CheckOrigin 来源校验，仅允许配置的域名连接
- **修复** DNS 事务 ID 从 `time.Now().UnixNano()` 改为 `math/rand/v2`
- **修复** CSV 解析器支持 `""` 转义引号序列
- **新增** 域名解析时校验标签长度（≤63 字符）和总长（≤253 字符）
- **修复** 发包域名选择从取模运算改为 `rand.IntN()`
- **移除** 死代码 `PCAPSender.ReadPCAPFile()`
- **新增** 前端测试框架（vitest + @testing-library/react），11 个测试用例

### v0.2.2 (2026-05-11) — QPS 批处理与 UDP 校验和

- **新增** 高 QPS（>2000）自动批处理，突破 `time.Sleep` 精度限制
- **新增** UDP 校验和实现（RFC 768 伪头部），替代原先恒为 0 的校验和
- **新增** 34 个测试用例（总计 57 个 Go 测试），覆盖 engine/scheduler/store
- **新增** `RedisOps` 接口提升 scheduler 可测试性
- **修复** 首次启动 schema 迁移可能失败的问题（ALTER 错误被忽略）

### v0.2.1 (2026-05-09) — 任务编辑与实时推送

- **新增** 任务编辑功能（EditTaskModal），非运行状态下可修改所有配置
- **新增** WebSocket 实时推送每包统计（已发送、已失败、运行时长）
- **新增** 任务时间追踪（created_at、last_run_at、total_run_ms）
- **新增** file_path 可编辑
- **新增** Redis 状态存储层

### v0.2.0 (2026-05-09) — PCAP 回放引擎

- **新增** PCAP 抓包文件回放模式，基于 gopacket 解析 + AF_PACKET raw socket
- **新增** 以太网帧地址自动改写（源/目的 MAC + IP 替换，校验和重算）
- **新增** PCAP 目录浏览器（支持按日期组织的子目录）
- **新增** 任务文件下载功能
- **新增** 可配置端口（`.env` 文件）
- **新增** host 网络模式部署
- **新增** 任务运行时长跟踪

### v0.1.0 (2026-05-08) — 初始版本

- CSV 域名列表输入模式，UDP socket DNS 发包
- QoS 控制（目标 QPS、抖动、延迟）
- WebSocket 实时监控面板
- REST API（任务 CRUD + 启动/停止/统计）
- React 18 + Ant Design 5 Web 控制台
- Docker Compose 一键部署
- SQLite 任务持久化

## License

MIT
