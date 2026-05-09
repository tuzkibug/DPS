# DPS — DNS Packet Sender

高性能 DNS 请求发包机，支持 CSV 域名列表和 PCAP 抓包文件两种输入模式，通过 raw socket 发送 DNS 查询包，可灵活控制发包速率、抖动和延迟。

## 功能特性

- **双输入模式**：CSV 域名列表 / PCAP 抓包文件回放（以太网帧级地址改写）
- **PCAP 服务器路径**：无需上传，直接选择服务器端 PCAP 目录或文件，支持按日期组织
- **QoS 控制**：精细控制发包速率（QPS）、抖动（Jitter）、延迟（Delay）
- **实时监控**：WebSocket 推送统计（当前 QPS、失败数、运行时长、累计运行时长）
- **任务编辑**：非运行状态下可修改任务配置（名称、路径、IP/MAC、QoS 参数）
- **实时推送**：WebSocket 每秒推送统计，监控数据动态更新无需刷新页面
- **Web 控制台**：React + Ant Design 现代前端界面，PCAP 目录浏览器
- **可配置端口**：通过 `.env` 文件自定义所有端口，适配任意部署环境

## 项目架构

```
┌─────────────────────────────────────────────────────────────────┐
│                     Frontend (React 18 + TypeScript)              │
│  Pages: TaskList / TaskCreate / TaskDetail                       │
│  Components: TaskForm / TaskList / LiveMonitor                   │
│  State: Zustand                                                  │
│  UI: Ant Design 5 + Recharts                                     │
└──────────────────────────────┬──────────────────────────────────┘
                               │ HTTP REST + WebSocket
                               │ (nginx reverse proxy → Gin)
┌──────────────────────────────▼──────────────────────────────────┐
│                     Backend (Go 1.22 + Gin)                       │
│                                                                   │
│  ┌─────────────┐   ┌──────────────┐   ┌─────────────────────┐   │
│  │  api/handler │   │  scheduler   │   │  engine             │   │
│  │  task.go     │   │  scheduler   │   │  dns.go   (packet)  │   │
│  │  websocket   │──▶│  .go         │──▶│  sender.go (socket) │   │
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
| **前端** | React 18, TypeScript, Vite 5, Ant Design 5, Zustand, Recharts, react-router-dom v6 |
| **后端** | Go 1.22, Gin, gorilla/websocket, google/uuid |
| **存储** | SQLite (任务持久化), Redis (实时状态与统计) |
| **发包** | Go `net.ListenPacket` (UDP raw socket) |
| **实时推送** | WebSocket |
| **部署** | Docker + Docker Compose |

### 目录结构

```
DPS/
├── backend/
│   ├── cmd/server/main.go            # 程序入口，路由注册
│   └── internal/
│       ├── api/handler/
│       │   ├── task.go               # REST API (CRUD + start/stop/stats + file download)
│       │   └── websocket.go          # WebSocket 实时推送
│       ├── engine/
│       │   ├── dns.go                # DNS 包构造、域名解析、QoS 控制器
│       │   ├── dns_test.go
│       │   ├── pcap.go               # PCAP 解析、包地址改写、raw socket 发送
│       │   └── sender.go             # UDP 发包器 + PCAP 回放器
│       ├── scheduler/
│       │   └── scheduler.go          # 任务调度，生命周期管理
│       └── store/
│           ├── sqlite.go             # SQLite 持久化
│           ├── sqlite_test.go
│           └── redis.go              # Redis 实时状态
├── web-ui/
│   ├── nginx.conf                    # 生产 Nginx 配置（含 API 代理 + WS 升级）
│   ├── vite.config.ts                # Vite 配置（开发代理）
│   └── src/
│       ├── App.tsx                    # 路由配置
│       ├── main.tsx                   # 入口
│       ├── api/
│       │   ├── index.ts              # Axios 实例 + API 方法
│       │   └── types.ts              # TypeScript 类型定义
│       ├── components/
│       │   ├── TaskForm.tsx           # 任务创建表单
│       │   ├── TaskList.tsx           # 任务列表表格
│       │   └── LiveMonitor.tsx        # 实时监控面板（WebSocket）
│       ├── pages/
│       │   ├── TaskListPage.tsx       # 任务列表页
│       │   ├── TaskCreatePage.tsx     # 创建任务页
│       │   └── TaskDetailPage.tsx     # 任务详情页（含配置展示与文件下载）
│       └── stores/
│           └── taskStore.ts           # Zustand 状态管理
├── docker-compose.yml                # 三服务编排
└── domain_list_example.csv           # CSV 示例文件
```

## 快速开始

### 环境要求

- Docker & Docker Compose
- （本地开发）Go 1.22+, Node.js 18+

### Docker 部署

```bash
# 克隆项目
git clone <repo-url>
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

> Backend 和 Frontend 使用 `network_mode: host` 共享宿主机网络栈，
> 确保 raw socket (AF_PACKET) 可直接绑定物理网卡发送 PCAP 回放流量。

#### 自定义端口

若默认端口被占用，创建 `.env` 文件（参考 `.env.example`）：

```
REDIS_PORT=6380
BACKEND_PORT=9090
FRONTEND_PORT=4000
```

`docker compose up -d` 会自动读取。不改端口则无需创建 `.env`。

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

**PCAP 模式**：将 PCAP 文件放入 `./pcap/` 目录（Docker 自动挂载），可按日期组织子目录，例如：

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
| Target QPS | 目标每秒发包数 | `100` |
| Jitter | 速率抖动比例 (0-1) | `0` |
| Min/Max Delay | 每包额外延迟范围 (ms) | `0` / `0` |

> PCAP 模式会将包内原有的源/目的 MAC 和 IP 全部替换为配置值，重算校验和后通过 raw socket 发送完整以太网帧。

### 3. 操作任务

- **查看列表**：Tasks 页面展示所有任务，含名称、类型、目标 IP、状态
- **点击任务名**：进入详情页，查看完整配置；CSV 任务可下载上传文件
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

监控数据通过 WebSocket 推送，刷新页面后状态保持。

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
# 创建任务（不含文件）
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "DNS Benchmark",
    "input_type": "csv",
    "src_ip": "10.0.2.15",
    "dst_ip": "8.8.8.8",
    "src_mac": "aa:bb:cc:dd:ee:ff",
    "dst_mac": "11:22:33:44:55:66",
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
| `target_qps` | 目标每秒发包数 | 控制发包间隔 = 1s / QPS，如 100 QPS → 每 10ms 发一包 |
| `jitter` | 速率抖动 (0-1) | 在发包间隔上叠加 ±jitter% 的随机偏差，模拟真实流量波动 |
| `delay_min_ms` | 最小额外延迟 | 每包固定额外延迟（与 delay_max 配合生成随机延迟） |
| `delay_max_ms` | 最大额外延迟 | 额外延迟 = random[min, max]，模拟网络延迟场景 |

## 运行测试

```bash
cd backend
go test ./... -v
```

## 注意事项

1. Backend 使用 host 网络模式 + `NET_ADMIN` + `NET_RAW`，确保 raw socket 可绑定物理网卡
2. CSV 模式使用标准 UDP socket，内核处理路由；PCAP 模式使用 AF_PACKET raw socket，直接发送以太网帧
3. 确保目标 DNS 服务器的 UDP 53 端口可达
4. 高 QPS 发包会消耗大量 CPU 和网络带宽，请酌情使用
5. 后端重启后，正在运行的任务将丢失（状态存储在内存中），需重新 Start

## License

MIT
