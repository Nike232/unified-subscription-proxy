# Operations Runbook

## Workflow Meanings

- `CI`
  快速反馈，默认必须关注
- `Docker Build & Push`
  `main` 上只做构建校验；`tag` 或手动触发时才推送镜像
- `Integration`
  仅手动触发，用于 `rust / dual` 重型联调

## Health Checks

核心检查接口：

- `/healthz`
- `/api/public/catalog`
- `/api/admin/kernel-status`

推荐检查顺序：

1. `control-plane /healthz`
2. `web /api/public/catalog`
3. 管理员登录后查看 `/api/admin/kernel-status`

## First-Line Troubleshooting

### 账号池问题

优先检查：

- 后台账号状态
- 最近健康检查错误
- 最近刷新错误
- 冷却状态与连续失败次数

### 支付问题

优先检查：

- 用户订单状态
- `mockpay` 回调是否成功
- Payment webhook 返回值
- 订阅是否创建或续期

### API Key 问题

优先检查：

- API Key 是否为 `active`
- 订阅是否过期
- 套餐是否允许访问对应模型 alias
- usage log 中的错误类型

## Heavy Integration

Rust 与双栈联调不再自动跑，改成手动触发：

1. 打开 GitHub Actions
2. 选择 `Integration`
3. 点击 `Run workflow`

只有在需要验证：

- `proxy-core-rust`
- `dual` 主用切换
- Rust compose 启动链

时才建议触发。

## First Production Validation

推荐固定执行以下流程：

1. 管理员登录控制面
2. 检查 `/api/admin/kernel-status`
3. 用户登录
4. 创建订单
5. 完成 `mockpay` 回调
6. 查看订阅和 API Key
7. 使用 API Key 调用 `proxy-core-go`
8. 查看 usage log
