# GrokForge

单机账号注册与凭证管理控制面（管理台 + 任务编排）。

> 仅供自用研究与授权测试。请遵守当地法律与上游服务条款。仓库与镜像不包含任何真实密钥、代理或账号数据。

## 默认端口

`17890`（`GROKFORGE_HTTP_ADDR` 可覆盖）

## 本地开发（默认路径）

**本机二进制 + 已有 PostgreSQL；Redis 可选。不要为日常开发 build 应用镜像。**

```bash
cp .env.example .env
# 编辑：
#   DATABASE_URL   → 指向你已有的 Postgres（库名建议 grokforge）
#   GROKFORGE_ENV=dev
#   GROKFORGE_MASTER_KEY / JWT（≥16 字符）
#   REDIS_URL      → 可先不填（管理面可用；创建任务会 503）
# 切勿提交 .env

export GOTOOLCHAIN=local
make frontend-embed backend   # 或仅 make backend（若已 embed 过前端）
set -a && source .env && set +a
export GROKFORGE_CLOAK_DRY_RUN=1   # 无 Cloak 二进制时
./backend/bin/grokforge
# http://127.0.0.1:17890  → 安装向导设密 → 登录
```

| 能力 | 无 Redis（dev） | 有 Redis |
|------|-----------------|----------|
| 安装向导 / 登录 / 账号 / 代理 | ✅ | ✅ |
| 创建任务 / Worker | ❌ `REDIS_UNAVAILABLE` | ✅ |
| `/readyz` | PG 好则 ready | PG + Redis |

需要任务队列时再起一个 Redis（示例）：

```bash
make redis-dev
# .env 中取消注释 REDIS_URL=redis://127.0.0.1:26379/0 后重启进程
```

可选：隔离沙箱数据服务（非默认日常路径）：

```bash
docker compose up -d   # 示例 Postgres :25432 + Redis :26379
```

## 生产 / 发布（镜像）

应用打包为**一个容器**（API + Worker + 前端静态）；PostgreSQL / Redis **分离**（可复用已有实例）。

镜像：`ghcr.io/zninggo/grokforge`

```bash
# 本机构建镜像（发布用，非日常开发）
make image

# 或 compose 全栈示例
cp .env.example .env   # 至少填 MASTER_KEY；生产勿用 dev 省 Redis
docker compose -f docker-compose.app.yml up -d --build
```

有 GHCR 权限时：

```bash
docker pull ghcr.io/zninggo/grokforge:latest
docker run -d --name grokforge -p 17890:17890 --env-file .env \
  ghcr.io/zninggo/grokforge:latest
```

## 主要能力

| 能力 | 说明 |
|------|------|
| 安装向导 | 强制管理员设密 |
| 任务 | noop / register / probe，Redis 队列 |
| 代理池 | 静态导入，脱敏展示 |
| YYDS | 临时邮箱 + 自动 OTP（300s） |
| 账号 | 加密存储、导出审计、本地凭证测活 |
| 门禁 | `make check-no-chat` 禁止聊天兼容 API 路径 |

## 安全

- 不要将 `.env`、代理密码、API Key、注册产物提交到 git
- 首次启动通过安装向导重置密码
- **不提供** OpenAI/Anthropic 聊天兼容推理 API
- 公开仓仅保留占位配置；真实密钥只放本机 `.env`

## 健康检查

- `GET /healthz` 存活
- `GET /readyz` 就绪（Postgres 必查；生产默认亦查 Redis。`GROKFORGE_ENV=dev` 且未配 Redis 时 Redis 不挡 ready）

## 许可证

见 [LICENSE](./LICENSE)。
