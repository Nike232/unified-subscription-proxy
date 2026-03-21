# Production Deployment

当前默认生产方案固定为：

- `control-plane`
- `proxy-core-go`
- `web`
- `postgres`

`proxy-core-rust` 继续保留，但仅作为实验/手动验证链，不进入默认上线方案。

## Server Prerequisites

- Linux 服务器
- Docker Engine 与 Docker Compose Plugin
- 可用域名两组：
  - `app.example.com` 指向 `web`
  - `api.example.com` 指向 `control-plane`
- 可写磁盘卷用于：
  - PostgreSQL 数据目录
  - `control-plane` 本地数据目录

## Environment Files

1. 复制生产模板：

```bash
cp .env.production.example .env.production
```

2. 至少填写这些字段：

- `CONTROL_PLANE_ORIGIN`
- `CONTROL_PLANE_PUBLIC_ORIGIN`
- `DATABASE_URL`
- `POSTGRES_USER`
- `POSTGRES_PASSWORD`
- `POSTGRES_DB`
- `ADMIN_BOOTSTRAP_PASSWORD`
- provider 凭据
- OAuth 回调地址

3. 默认保持：

- `PROXY_CORE_MODE=go`
- `PROXY_CORE_PRIMARY=go`

## Start Commands

推荐使用生产覆盖文件：

```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

查看状态：

```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml ps
```

查看日志：

```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml logs -f control-plane proxy-core-go web db
```

## First Initialization

首次启动后验证：

```bash
curl -fsS https://api.example.com/healthz
curl -fsS https://api.example.com/api/public/catalog
```

默认演示账号仅用于联调，生产前应至少修改：

- 管理员密码
- 用户默认密码

## Reverse Proxy

建议反代规则：

- `app.example.com` -> `web`
- `api.example.com` -> `control-plane`

要求：

- 保留 `X-Forwarded-For`
- 保留 `X-Forwarded-Proto`
- HTTPS 终止后转发到容器内部 HTTP

## Persistence

必须持久化：

- PostgreSQL 数据卷
- `control-plane` 数据目录

建议对 PostgreSQL 做定期备份；如果只使用数据库作为主存储，恢复时优先恢复数据库，再恢复应用容器。

## Upgrade

日常升级流程：

1. 拉取新版本镜像或新代码
2. 备份 PostgreSQL
3. 执行 `docker compose ... up -d`
4. 检查 `/healthz`、`/api/public/catalog`
5. 用管理员和普通用户各做一次最小验收

## Rollback

回滚流程：

1. 切回上一个镜像 tag 或上一版代码
2. `docker compose ... up -d`
3. 如发生数据兼容问题，再恢复最近数据库备份
