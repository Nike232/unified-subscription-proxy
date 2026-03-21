# Unified Subscription Proxy

一个面向 `Claude / Codex / OpenAI(GPT) / Gemini / Antigravity` 的统一订阅反代平台 monorepo。

当前仓库采用双栈内核并行，但当前默认生产主链已经固定为 `proxy-core-go`：

- `apps/proxy-core-go`
  Go 实现的统一反代内核
- `apps/proxy-core-rust`
  Rust 实现的高集成反代内核

当前版本实现了一个可运行的订阅代理平台：

- `apps/control-plane`
  用户、订阅、套餐、API Key、上游账号池的控制面 API
- `apps/proxy-core-go`
  当前默认生产反代内核
- `apps/web`
  管理后台静态页，直接消费控制面 API

当前已完成的基础改造：

- `control-plane` 支持 `file` 和 `postgres` 两种存储后端
- `proxy-core` 不再直接读本地数据文件，而是通过 HTTP 调用控制面内部接口获取平台快照并回写 usage log

## Run

当前建议把运行方式理解为三层：

- 本地开发：直接运行 Go 控制面和 Go 内核
- 默认生产部署：`control-plane + proxy-core-go + web + postgres`
- 手动双栈验证：仅在需要时单独验证 `proxy-core-rust`

```bash
cd /Users/tomfng/projects/claw_test/zhongzhuan/unified-subscription-proxy
cp .env.example .env
go run ./apps/control-plane
go run ./apps/proxy-core-go
```

如果要跑 PostgreSQL 版本：

```bash
docker compose up -d db
CONTROL_PLANE_STORE_BACKEND=postgres \
DATABASE_URL='postgres://postgres:postgres@127.0.0.1:5432/unified_subscription_proxy?sslmode=disable' \
go run ./apps/control-plane
```

打开：

- Control Plane: `http://127.0.0.1:8080`
- Proxy Core Go: `http://127.0.0.1:8081`
- Proxy Core Rust: `http://127.0.0.1:8045`
- Admin Web: `http://127.0.0.1:3000/`

## Runtime Selection

- `PROXY_CORE_MODE=go|rust|dual`
- `PROXY_CORE_PRIMARY=go|rust`
- `PROXY_CORE_GO_ORIGIN` 和 `PROXY_CORE_RUST_ORIGIN` 控制两条内核的公开地址
- `CONTROL_PLANE_PUBLIC_ORIGIN` 控制用户支付回跳和 checkout 链接生成地址
- `WEB_CONTROL_PLANE_UPSTREAM` 只用于 `web` 容器内的 `/api` 与 `/mockpay` 反向代理
- 默认生产口径使用：
  - `PROXY_CORE_MODE=go`
  - `PROXY_CORE_PRIMARY=go`
- `control-plane` 的 `/api/public/catalog` 和 `/api/admin/kernel-status` 会返回当前主用内核与双栈健康状态

## Compose Modes

- `docker compose up -d --build db control-plane web proxy-core-go` 对应 `PROXY_CORE_MODE=go`
- `docker compose up -d --build db control-plane web proxy-core-rust` 对应 `PROXY_CORE_MODE=rust`
- `docker compose up -d --build db control-plane web proxy-core-go proxy-core-rust` 对应 `PROXY_CORE_MODE=dual`
- 当前 `CI` 默认只跑 `go` 最小联调
- `Integration` 保留 `rust / dual` 重型联调，但改为手动触发

## Production Deployment

默认生产部署建议只起这四个服务：

- `db`
- `control-plane`
- `proxy-core-go`
- `web`

生产部署文件：

- 环境模板：[.env.production.example](/Users/tomfng/projects/claw_test/zhongzhuan/unified-subscription-proxy/.env.production.example)
- 生产 compose 覆盖：[docker-compose.prod.yml](/Users/tomfng/projects/claw_test/zhongzhuan/unified-subscription-proxy/docker-compose.prod.yml)
- 部署文档：[deploy.md](/Users/tomfng/projects/claw_test/zhongzhuan/unified-subscription-proxy/docs/deploy.md)
- 运维手册：[operations.md](/Users/tomfng/projects/claw_test/zhongzhuan/unified-subscription-proxy/docs/operations.md)

生产启动示例：

```bash
cp .env.production.example .env.production
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

## Key Ideas

- 控制面与反代内核分离，但共享统一数据模型
- 用户只拿统一 API Key，不接触上游账号
- 套餐定义可访问的 provider、模型别名和 fallback 策略
- 账号池按 provider 共享调度，支持优先级、权重和健康状态

## Current Scope

这是一个可运行的 v0 实现，优先打通：

- 添加上游账号
- 配置服务套餐
- 分配订阅
- 创建 API Key
- 通过统一模型别名发起请求并得到调度结果

当前已具备真实执行能力：

- `OpenAI` 与 `Gemini` 支持真实请求转发
- `Claude` 支持 OpenAI 风格输入到 Anthropic Messages 的转换
- `Codex` 作为独立 provider 接入，并支持在 `gpt-reasoning` 路由上的 fallback
- `Antigravity` 支持通过 `hybrid-premium` 真实执行，并在失败时回退到 `Gemini`

当前仍未完成：

- `Responses API` / WebSocket 扩展入口

目前与最初版本相比，已经额外完成：

- 控制面存储后端抽象，可切换 PostgreSQL
- 控制面内部接口：平台快照、usage log 回写
- `proxy-core` 与 `control-plane` 的 HTTP 边界
- provider 健康检查、usage log 查询、alias 路由调试
- `OpenAI -> Codex` 运行时 fallback
- `Antigravity -> Gemini` 运行时 fallback
- OAuth 登录会话、回调入口、手动刷新入口
- 控制面内置自动任务：预刷新、健康巡检、OAuth 会话清理
- 账号健康治理：连续失败计数、冷却窗口、手动恢复、手动解除冷却
- usage log 支持按 `error_type` 和 `account_id` 筛选
- 用户/管理员登录态与角色隔离
- 用户自助下单、Mock 支付回调、订阅生效与 API Key 自助管理
- 双栈运行时配置、内核健康探测与最小共享契约测试

## Build Notes

- `deploy/` 下的旧 Dockerfile 已不再作为主构建入口；当前以 `apps/*/Dockerfile` 和 GitHub Actions matrix 为准
- `CI`：快速反馈，默认包含 Go、Rust、Web 和 Go 最小 smoke
- `Integration`：仅手动触发，用于 Rust/Dual 重型联调
- `Docker Build & Push`：`main` 上只 build；只有 `tag` 或手动触发时才推送镜像
- Docker 发布会分别产出：
  - `control-plane`
  - `proxy-core-go`
  - `proxy-core-rust`
  - `web`

## Demo Accounts

- 管理员：`admin@example.com` / `admin123`
- 普通用户：`demo@example.com` / `demo123`

## Payment Flow

- 用户侧先创建订单
- 控制面创建 `mockpay` 支付记录并返回 checkout URL
- 访问 checkout URL 后点击 `Mark Paid`
- webhook 处理成功后自动创建或续期订阅，并按订单配置创建/绑定 API Key

## First Production Validation

首轮上线联调建议固定按这条路径执行：

1. 管理员登录控制面
2. 查看 `/api/admin/kernel-status`
3. 用户登录
4. 创建订单
5. 完成 `mockpay` 回调
6. 查看订阅和 API Key
7. 使用 API Key 调用 `proxy-core-go /v1/chat/completions`
8. 查看 usage log
