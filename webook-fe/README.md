# webook-fe

Next.js 前端：登录 / 个人资料 / 云盘（`/files`）。开发服务器 **:3000**，API 默认打本机后端 **:8080**。

## 本地开发

后端先在 `webook/` 里 `go run .`（监听 8080）。然后：

```bash
cd webook-fe
npm install
unset NEXT_PUBLIC_API_BASE_URL
npm run dev
```

打开 [http://localhost:3000](http://localhost:3000)。登录后个人信息页 → **我的云盘**。

未设置 `NEXT_PUBLIC_API_BASE_URL` 时，axios 使用 `当前主机:8080`。不要设成 `:88`（那是 K8s Service 口）。

## K8s 发版

流水线：宿主机 `npm run build`（standalone）→ 镜像只拷产物 → 导入 kind → 滚动更新。容器内不跑 `npm ci`。

```bash
cd webook-fe
make help
make release VERSION=v0.0.2
```

验收用 **http://live.webook.com** 或 **http://\<宿主机IP\>:89**，不要用 `:3000`。

完整命令、变量、排错见 [Makefile使用指南.md](./Makefile使用指南.md)。全栈（含后端）见仓库根目录 README 和 `webook/docs/Makefile使用指南.md`。
