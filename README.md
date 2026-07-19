# GrokForge

单机账号注册与凭证管理控制面（管理台 + 任务编排）。

> 仅供自用研究与授权测试。请遵守当地法律与上游服务条款。仓库与镜像不包含任何真实密钥、代理或账号数据。

## 默认端口

`17890`（`GROKFORGE_HTTP_ADDR` 可覆盖）

## 部署形态

- **一个应用容器**：API + Worker + 前端静态资源（Cloak 二进制可选挂载）
- **PostgreSQL / Redis 分离**（可复用已有实例）

镜像：`ghcr.io/zninggo/grokforge`

## 快速开始

### 1) 仅数据依赖（开发）

```bash
cp .env.example .env
# 编辑 DATABASE_URL / REDIS_URL / GROKFORGE_MASTER_KEY / GROKFORGE_YYDS_API_KEY

docker compose up -d          # postgres :25432, redis :26379
export GOTOOLCHAIN=local
make frontend-embed backend
set -a && source .env && set +a
export GROKFORGE_CLOAK_DRY_RUN=1   # 无 Cloak 二进制时
./backend/bin/grokforge
# http://127.0.0.1:17890  → 安装向导设密 → 登录
```

### 2) 应用镜像（本机构建）

```bash
cp .env.example .env   # 至少填 MASTER_KEY；YYDS 可选
docker compose -f docker-compose.app.yml up -d --build
# http://127.0.0.1:17890
```

### 3) 拉取镜像运行（有 GHCR 权限时）

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

## 健康检查

- `GET /healthz` 存活
- `GET /readyz` 就绪（PG + Redis）

## 许可证

见 [LICENSE](./LICENSE)。
