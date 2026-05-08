# DPS — DNS Packet Sender

高性能 DNS 请求发包机，支持 CSV 域名列表和 PCAP 抓包文件两种输入模式，通过 raw socket 发送 DNS 查询包，可灵活控制发包速率、抖动和延迟。

## 功能特性

- **双输入模式**：CSV 域名列表 / PCAP 抓包文件回放
- **QoS 控制**：精细控制发包速率（QPS）、抖动（Jitter）、延迟（Delay）
- **实时监控**：WebSocket 推送实时发包统计（发送量、当前 QPS、失败数、运行时长）
- **任务管理**：创建、查看配置、启动、停止、删除任务，支持上传文件下载
- **Web 控制台**：React + Ant Design 现代前端界面

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
│       │   └── sender.go             # UDP 发包器
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

### 1. 准备域名列表

创建 CSV 文件，每行一个域名：

```
www.example.com
www.google.com
dns.example.org
```

项目根目录提供了 `domain_list_example.csv` 作为参考。

### 2. 创建任务

访问 http://localhost:3000 → 点击 **Create Task**，填写配置：

| 字段 | 说明 | 示例 |
|------|------|------|
| Task Name | 任务名称 | `DNS压力测试` |
| Input Type | CSV 或 PCAP | `csv` |
| Upload File | 上传域名列表文件 | `domains.csv` |
| Source IP | 源 IP 地址 | `10.0.2.15` |
| Destination IP | 目标 DNS 服务器 IP | `8.8.8.8` |
| Source MAC | 源 MAC 地址 | `aa:bb:cc:dd:ee:ff` |
| Destination MAC | 目标 MAC 地址 | `11:22:33:44:55:66` |
| Target QPS | 目标每秒发包数 | `100` |
| Jitter | 速率抖动比例 (0-1)，在 QPS 间隔上叠加随机偏差 | `0.1` |
| Min/Max Delay | 每包额外延迟范围 (ms)，模拟网络延迟 | `0` / `0` |

### 3. 操作任务

- **查看列表**：Tasks 页面展示所有任务，含名称、类型、目标 IP、状态
- **点击任务名**：进入详情页，查看完整配置并可下载上传的文件
- **Start**：启动发包，状态变为 `running`
- **Stop**：停止发包，状态恢复 `pending`
- **Delete**：删除任务及关联的上传文件

### 4. 实时监控

任务启动后，实时面板展示：

- **Sent Packets**：累计发包数
- **Current QPS**：当前实际每秒发包数
- **Failed**：发包失败数
- **Elapsed**：已运行时间

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

1. 发包需要 root 权限或 `NET_RAW` capability，Docker 部署已配置 `NET_ADMIN` + `NET_RAW`
2. 确保目标 DNS 服务器的 UDP 53 端口可达
3. 高 QPS 发包会消耗大量 CPU 和网络带宽，请酌情使用
4. 后端重启后，正在运行的任务将丢失（状态存储在内存中），需重新 Start

## License

MIT
