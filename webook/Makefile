# 文件别名 docker
.PHONY: docker
docker:
	@rm webook || true
	#	-f 强制删除
	@docker rmi -f clould/webook:v0.0.1
	@go mod tidy
	# 编译项目文件 到arm架构的linux上 名字叫做webook
	@GOOS=linux GOARCH=arm go build -o webook .
	@docker build -t clould/webook:v0.0.1 .
