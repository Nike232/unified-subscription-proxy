# Unified Subscription Proxy

一个面向 `Claude / Codex / OpenAI(GPT) / Gemini / Antigravity` 的统一订阅反代平台 monorepo。

当前版本实现了一个可运行的首版骨架：

- `apps/control-plane`
  用户、订阅、套餐、API Key、上游账号池的控制面 API
- `apps/proxy-core`
  模型别名路由、账号池调度、API Key 校验、真实 provider 执行
- `apps/web`
  管理后台静态页，直接消费控制面 API

当前已进入下一阶段基础改造：

- `control-plane` 支持 `file` 和 `postgres` 两种存储后端
- `proxy-core` 不再直接读本地数据文件，而是通过 HTTP 调用控制面内部接口获取平台快照并回写 usage log

## Run

```bash
cd /Users/tomfng/projects/claw_test/zhongzhuan/unified-subscription-proxy
cp .env.example .env
go run ./apps/control-plane
go run ./apps/proxy-core
```

如果要跑 PostgreSQL 版本：

```bash
docker compose up -d postgres
CONTROL_PLANE_STORE_BACKEND=postgres \
DATABASE_URL='postgres://postgres:postgres@127.0.0.1:5432/unified_subscription_proxy?sslmode=disable' \
go run ./apps/control-plane
```

打开：

- Control Plane: `http://127.0.0.1:8080`
- Proxy Core: `http://127.0.0.1:8081`
- Admin Web: `http://127.0.0.1:8080/`

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

## Demo Accounts

- 管理员：`admin@example.com` / `admin123`
- 普通用户：`demo@example.com` / `demo123`

## Payment Flow

- 用户侧先创建订单
- 控制面创建 `mockpay` 支付记录并返回 checkout URL
- 访问 checkout URL 后点击 `Mark Paid`
- webhook 处理成功后自动创建或续期订阅，并按订单配置创建/绑定 API Key
