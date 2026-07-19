# GrokForge

单机账号注册与凭证管理控制面（管理台 + 任务编排）。

> 仅供自用研究与授权测试。请遵守当地法律与上游服务条款。仓库与镜像不包含任何真实密钥、代理或账号数据。

## 默认端口

`17890`（可用环境变量 `GROKFORGE_HTTP_ADDR` 覆盖）

## 部署形态

- **一个应用容器**：API + Worker + 前端静态资源 + 浏览器运行时
- **PostgreSQL / Redis 分离**（可复用已有实例）

镜像（构建后）：`ghcr.io/zninggo/grokforge`

## 快速开始（开发）

```bash
cp .env.example .env
# 编辑 .env：DATABASE_URL、REDIS_URL、GROKFORGE_MASTER_KEY、GROKFORGE_YYDS_API_KEY

make backend
# 后续 phase 补齐 migrate / frontend embed / compose
```

## 安全

- 不要将 `.env`、代理列表、API Key、注册产物提交到 git
- 首次启动请通过安装向导**重置管理员密码**
- 本项目**不提供** OpenAI/Anthropic 聊天兼容推理 API

## 许可证

见 [LICENSE](./LICENSE)。
