.PHONY: docker
docker:
	@rm webook || true
	@GOOS=linux GOARCH=amd64 go build -tags=k8s -o webook .
	@docker rmi -f could/webook:v0.0.1
	@docker build -t could/webook:v0.0.1 .