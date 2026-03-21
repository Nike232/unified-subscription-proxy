# Unified Subscription Proxy

一个面向 `Claude / Codex / OpenAI(GPT) / Gemini / Antigravity` 的统一订阅反代平台 monorepo。

当前版本实现了一个可运行的首版骨架：

- `apps/control-plane`
  用户、订阅、套餐、API Key、上游账号池的控制面 API
- `apps/proxy-core`
  模型别名路由、账号池调度、API Key 校验、模拟反代分发 API
- `apps/web`
  管理后台静态页，直接消费控制面 API

## Run

```bash
cd /Users/tomfng/projects/claw_test/zhongzhuan/unified-subscription-proxy
cp .env.example .env
go run ./apps/control-plane
go run ./apps/proxy-core
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

当前反代层使用模拟 dispatch 响应，便于先稳定控制面、数据模型和路由决策。真实 OAuth 刷新与上游请求执行可在这个骨架上继续迁入现有项目代码。

