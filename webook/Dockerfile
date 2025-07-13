# 基础镜像
FROM ubuntu:latest

# 把编译后的打包进这个镜像
COPY webook /app/webook
# 设定我的工作目录
WORKDIR /app

# 执行命令
ENTRYPOINT ["/app/webook"]

