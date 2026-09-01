# WeBook

Go + Next.js 示例：用户注册 / 登录、个人资料、云盘。后端按 DDD 分层，支持 Docker Compose 本地依赖与 Kubernetes 部署。

## 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go 1.22、Gin、GORM、JWT、Redis 限流 |
| 数据 | MySQL 8、Redis、本地文件存储 |
| 前端 | Next.js 13、React 18、Ant Design、TypeScript |
| 部署 | Docker Compose、Kubernetes + Ingress |

## 目录结构

```
webook/
├── webook/           # Go 后端（Gin，监听 :8080）
│   ├── config/       # 配置（dev / k8s build tag）
│   ├── internal/     # domain / repository / dao / service / web
│   ├── pkg/          # 公共中间件（限流等）
│   ├── script/mysql/ # 数据库初始化
│   ├── docker-compose.yaml
│   ├── Makefile      # 全栈 K8s 发版入口
│   └── k8s-*.yaml
├── webook-fe/        # Next.js 前端（开发 :3000）
│   ├── Makefile
│   ├── Makefile使用指南.md
│   └── k8s-*.yaml
└── go.mod
```

- 全栈 Makefile：[`webook/docs/Makefile使用指南.md`](webook/docs/Makefile使用指南.md)
- 前端 Makefile：[`webook-fe/Makefile使用指南.md`](webook-fe/Makefile使用指南.md)

## 本地开发

### 1. 启动依赖

```bash
cd webook
docker compose up -d
```

- MySQL：`localhost:13316`，root/root，库名 `webook`
- Redis：`localhost:6379`，无密码

### 2. 启动后端

```bash
cd webook
go run .
```

默认 `config/dev.go`（`!k8s`）：

- DSN：`root:root@tcp(localhost:13316)/webook`
- Redis：`localhost:6379`
- HTTP：**http://localhost:8080**
- 文件：`./data/files`，分片：`./data/chunks`

### 3. 启动前端

```bash
cd webook-fe
npm install
unset NEXT_PUBLIC_API_BASE_URL   # 避免打到 K8s 的 :88
npm run dev
```

浏览器：**http://localhost:3000**。

未设置 `NEXT_PUBLIC_API_BASE_URL` 时，axios 默认：

- 浏览器：`当前协议 + 当前主机名 + :8080`（本机即 `http://localhost:8080`）
- SSR：`http://localhost:8080`

不要把开发前端指到 `:88`（那是 K8s Service 端口，本地 `go run` 不听这个口）。

登录后个人信息页有 **我的云盘**，进入 `/files`。

## 主要 API

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| POST | `/users/signup` | 注册 | 否 |
| POST | `/users/login` | 登录（JWT） | 否 |
| POST | `/users/edit` | 编辑资料 | 是 |
| GET | `/users/profile` | 个人资料 | 是 |
| GET | `/hello` | 健康检查 | 否 |
| GET | `/files?parentId=` | 文件列表 | 是 |
| POST | `/files/folder` | 新建文件夹 | 是 |
| GET | `/files/breadcrumbs?id=` | 面包屑 | 是 |
| POST | `/files/delete` | 删除（文件夹递归） | 是 |
| POST | `/files/upload` | 普通上传（multipart） | 是 |
| GET | `/files/download?id=` | 下载 | 是 |
| POST | `/files/sec-upload` | 秒传（按 MD5） | 是 |
| POST | `/files/chunk-upload` | 上传分片 | 是 |
| GET | `/files/chunk-upload?identifier=` | 已上传分片 | 是 |
| POST | `/files/merge` | 合并分片 | 是 |

JWT 响应头 `x-jwt-token`，请求带 `Authorization: Bearer <token>`。  
文件接口全部要登录，没有 IgnorePaths。

密码规则：至少 8 位，含字母、数字、特殊字符（`$@$!%*#?&`）。

## Kubernetes

后端镜像示例：`could/webook:v0.0.3`  
前端镜像示例：`could/webook-fe:v0.0.2`

K8s 用 `-tags=k8s`（`config/k8s.go`）：

- MySQL：`webook-mysql:11308`
- Redis：`webook-redis:10379`
- 文件：容器内 `/data/files`、`/data/chunks`（当前 Deployment 未挂卷，重启会丢文件）

### 端口

| | 开发 | K8s Service | Ingress |
|--|------|-------------|---------|
| 前端 | :3000 | :89 → Pod :3000 | `live.webook.com` |
| 后端 | :8080 | :88 → Pod :8080 | `api.webook.com` |

**验收集群不要用 `:3000`。** 用 `http://live.webook.com` 或 `http://<宿主机IP>:89`。  
kind LoadBalancer 的 `172.18.x.x` 在 Windows 宿主机上经常不通。

### 发版（推荐 Makefile）

```bash
cd webook

# 全新集群：基础设施 + Ingress + Deployment 骨架
make deploy-all-init

# 构建并滚动更新（前端会先在宿主机 npm run build）
make release-all VERSION=v0.0.2
```

前端镜像是 **宿主机 standalone 构建** 再 COPY 进镜像，容器内不跑 `npm ci`。  
生产前端的 API 地址构建时注入，默认 `http://api.webook.com`。

日常：

```bash
make release VERSION=v0.0.3      # 只发后端
make release-fe VERSION=v0.0.2   # 只发前端
make status-all
```

命令全表：[`webook/docs/Makefile使用指南.md`](webook/docs/Makefile使用指南.md)、[`webook-fe/Makefile使用指南.md`](webook-fe/Makefile使用指南.md)，或 `make help`。

### 清单（也可直接 kubectl）

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

hosts：

```
127.0.0.1 live.webook.com api.webook.com
```

## 架构要点

```
web → service → repository → dao / cache → MySQL / Redis
                    └── storage → 本地磁盘
```

- 登录校验：JWT 中间件；白名单 `/users/signup`、`/users/login`、`/hello`
- 限流：Redis，默认 1 秒 100 次
- CORS：`localhost`、含 `webook.com` 的源、以及私网 / 回环 IP
- 云盘：`files` / `user_files` / `file_chunks` 三张表；秒传按文件 MD5；无回收站 / 分享

## License

本仓库为学习 / 示例项目。
