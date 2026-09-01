# webook-fe Makefile 使用指南

本文档对应 **`webook-fe/Makefile`**。全栈（MySQL / Redis / 后端 / Ingress）仍在 `webook/` 目录用后端 Makefile 编排，见仓库 [`webook/docs/Makefile使用指南.md`](../webook/docs/Makefile使用指南.md)。

> 所有命令默认在 **`webook-fe/`** 下执行。

---

## 流水线（先搞清这个）

前端镜像 **不在容器里 `npm ci`**。Alpine / 代理把那条路搞挂过，已经删掉。

```
宿主机 npm run build（next.config.js: output standalone）
  → Docker 只拷 .next/standalone + .next/static + public
  → docker save | ctr import 进 kind 节点 desktop-control-plane
  → kubectl set image + rollout restart
```

镜像入口：`CMD ["node", "server.js"]`，容器端口 **3000**。

---

## 前置条件

| 工具 | 用途 |
|------|------|
| Node.js + npm | 宿主机 `npm run build`，发版必装 |
| Docker Desktop + Kubernetes | 打镜像、跑集群 |
| kubectl / make | 部署 |

第一次先 `npm install`，保证有 `node_modules/next`。

---

## 端口（别用错）

| 场景 | 地址 |
|------|------|
| 本地开发 | **http://localhost:3000**，API 默认 **:8080**（本机 `go run`） |
| K8s Service | **:89** → Pod :3000 |
| Ingress | **http://live.webook.com**，API **http://api.webook.com** |

**验收集群不要打开 `:3000`。** 那是 `npm run dev`，和 Pod 无关。

kind LoadBalancer 的 `EXTERNAL-IP`（常见 `172.18.0.2`）在 Windows 上经常不通。用 Ingress 或 `http://<宿主机IP>:89`。

---

## 本地开发（不走 Makefile）

```bash
cd webook-fe
npm install
unset NEXT_PUBLIC_API_BASE_URL
npm run dev
```

浏览器 http://localhost:3000。后端需已在 **:8080** 监听。  
设了 `NEXT_PUBLIC_API_BASE_URL=http://api.webook.com` 或 `:88`，本地开发会打到集群或打错端口。

---

## 快速发版

集群和 Deployment **已经存在** 时：

```bash
cd webook-fe
make release VERSION=v0.0.2
```

等价于 `make docker`（含宿主机 build + 导入 kind）+ `make deploy`。

从后端目录委托也可以：

```bash
cd webook
make release-fe VERSION=v0.0.2
```

默认 `API_BASE_URL=http://api.webook.com`（Ingress）。LoadBalancer 本机访问改：

```bash
make release VERSION=v0.0.2 API_BASE_URL=http://localhost:88
# 或从 webook/：
make release-fe VERSION=v0.0.2 FE_API_BASE_URL=http://localhost:88
```

`NEXT_PUBLIC_*` 编译进 JS，改变量必须重新 `make release`，只重启 Pod 无效。

---

## 变量

| 变量 | 默认 | 说明 |
|------|------|------|
| `IMAGE_NAME` | `could/webook-fe` | 镜像名 |
| `VERSION` | `v0.0.1` | 标签 |
| `API_BASE_URL` | `http://api.webook.com` | 构建时写入 axios |
| `KIND_NODE` | `desktop-control-plane` | kind 节点，用于 `ctr import` |
| `INGRESS_HOST` | `live.webook.com` | 前端域名 |
| `NAMESPACE` | `default` | |

---

## 命令

```bash
make help
```

| 命令 | 说明 |
|------|------|
| `make build` | 宿主机 `NEXT_PUBLIC_API_BASE_URL=... npm run build`，检查 `server.js` |
| `make docker` | build + `docker build` + `load-image` |
| `make load-image` | `docker save \| ctr -n k8s.io images import` |
| `make deploy` | `kubectl set image` + `rollout restart` |
| `make deploy-init` | 首次用 YAML 创建 Deployment |
| `make deploy-deps-init` | 创建 Service（:89） |
| `make deploy-ingress` | 应用 `k8s-ingress-fe.yaml` |
| `make release` | docker + deploy |
| `make status` / `make version` | 状态 / 当前镜像 |
| `make logs` / `make logs-follow` | 日志 |
| `make rollback` | 回滚 |
| `make stop-app` / `make start-app` | 副本 0 / 2 |
| `make destroy-app` | 删 Deployment + Service + Ingress |
| `make clean-docker` | 删本地该版本镜像 |

**deploy-init vs deploy：** 没有 Deployment 时用 `deploy-init`；已有则 `deploy` / `release`。

全栈首次（含 MySQL、后端）仍要：

```bash
cd webook
make deploy-all-init
make release-all VERSION=v0.0.1
```

---

## 常见问题

**`Missing dependencies; run: npm install`**  
没有 `node_modules/next`。

**`Missing .next/standalone/server.js`**  
`next.config.js` 必须 `output: 'standalone'`，且宿主机 build 成功。

**页面能开，登录 / 云盘 401 或连不上**  
1. 构建时 `API_BASE_URL` 和访问方式不一致  
2. 本地开发误带 `NEXT_PUBLIC_API_BASE_URL`  
3. 用 `:3000` 去验 K8s 镜像  

**同 tag 发版 Pod 还是旧页面**  
`make deploy` 已 `rollout restart`。确认镜像已 `load-image` 进节点。`make version` 看当前 image。

**Docker Desktop Containers 搜不到前端容器**  
Pod 在 `desktop-control-plane` 里面。看 Kubernetes 页或 `kubectl get pods -l app=webook-fe`。
