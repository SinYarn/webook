# WeBook

Go + Next.js 的 Web 应用示例：用户注册 / 登录、个人资料。后端按 DDD 分层，支持 Docker Compose 本地依赖与 Kubernetes 部署。

## 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go 1.22、Gin、GORM、JWT、Redis 限流 |
| 数据 | MySQL 8、Redis |
| 前端 | Next.js 13、React 18、Ant Design、TypeScript |
| 部署 | Docker Compose、Kubernetes + Ingress |

## 目录结构

```
webook/
├── webook/           # Go 后端（Gin API，监听 :8080）
│   ├── config/       # 配置（dev / k8s build tag）
│   ├── internal/     # domain / repository / service / web
│   ├── pkg/          # 公共中间件（限流等）
│   ├── script/mysql/ # 数据库初始化
│   ├── docker-compose.yaml
│   └── k8s-*.yaml
├── webook-fe/        # Next.js 前端（:3000）
│   └── k8s-*.yaml
└── go.mod
```

## 本地开发

### 1. 启动依赖

```bash
cd webook
docker compose up -d
```

会起来：

- MySQL：`localhost:13316`，root/root，库名 `webook`
- Redis：`localhost:6379`，无密码

### 2. 启动后端

```bash
cd webook
go run .
```

默认使用 `config/dev.go`（`!k8s` build tag）：

- DSN：`root:root@tcp(localhost:13316)/webook`
- Redis：`localhost:6379`
- HTTP：`http://localhost:8080`

### 3. 启动前端

```bash
cd webook-fe
npm install
npm run dev
```

浏览器打开 [http://localhost:3000](http://localhost:3000)。

API 地址默认 `http://localhost:88`（见 `src/axios/axios.ts`）。本地若直接打后端，可设：

```bash
export NEXT_PUBLIC_API_BASE_URL=http://localhost:8080
npm run dev
```

## 主要 API

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| POST | `/users/signup` | 注册 | 否 |
| POST | `/users/login` | 登录（JWT） | 否 |
| POST | `/users/edit` | 编辑资料 | 是 |
| GET | `/users/profile` | 个人资料 | 是 |
| GET | `/hello` | 健康检查 | 否 |

JWT 通过响应头 `x-jwt-token` 下发，请求需带 `Authorization: Bearer <token>`。

## Kubernetes

后端镜像示例：`could/webook:v0.0.1`  
前端镜像示例：`could/webook-fe:v0.0.1`

K8s 配置使用 `-tags=k8s` 编译（`config/k8s.go`），连接：

- MySQL：`webook-mysql:11308`
- Redis：`webook-redis:10379`

典型资源：

```bash
# 后端 + 依赖
kubectl apply -f webook/k8s-mysql-pv.yaml
kubectl apply -f webook/k8s-mysql-pvc.yaml
kubectl apply -f webook/k8s-mysql-deployment.yaml
kubectl apply -f webook/k8s-mysql-service.yaml
kubectl apply -f webook/k8s-redis-deployment.yaml
kubectl apply -f webook/k8s-redis-service.yaml
kubectl apply -f webook/k8s-webook-deployment.yaml
kubectl apply -f webook/k8s-webook-service.yaml
kubectl apply -f webook/k8s-ingress-nginx.yaml

# 前端
kubectl apply -f webook-fe/k8s-webook-fe-deployment.yaml
kubectl apply -f webook-fe/k8s-webook-fe-service.yaml
kubectl apply -f webook-fe/k8s-ingress-fe.yaml
```

Ingress 默认域名：

- API：`api.webook.com` → Service `webook:88`
- 前端：`live.webook.com` → Service `webook-fe:89`

本地 hosts 自行映射，或改 Ingress host。

## 架构要点

后端分层（由外到内）：

```
web → service → repository → dao / cache → MySQL / Redis
```

- 登录校验：JWT 中间件，白名单 `/users/signup`、`/users/login`、`/hello`
- 限流：Redis，默认 1 秒 100 次
- CORS：允许 `localhost` 与 `webook.com` 源

## License

本仓库为学习 / 示例项目。
