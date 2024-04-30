# 文件别名 docker
.PHONY: docker
docker:
	@rm webook || true
	@GOOS=linux GOARCH=arm go build -o webook .
	#	-f 强制删除
	@docker rmi -f clould/webook:v0.0.1

	@docker build -t clould/webook:v0.0.1 .
