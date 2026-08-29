# Frontend release pipeline: host build -> standalone image -> kind -> Kubernetes
IMAGE_NAME ?= could/webook-fe
VERSION ?= v0.0.1
NAMESPACE ?= default
DEPLOYMENT_NAME ?= webook-fe
DEPLOYMENT_FILE ?= k8s-webook-fe-deployment.yaml
INGRESS_NAMESPACE ?= ingress-nginx
INGRESS_HOST ?= live.webook.com
# 构建时注入：Ingress 模式用 http://api.webook.com，LoadBalancer 模式用 http://localhost:88
API_BASE_URL ?= http://api.webook.com
KIND_NODE ?= desktop-control-plane

.DEFAULT_GOAL := help

.PHONY: help
help:
	@echo "webook-fe K8s Deployment Commands:"
	@echo "  make build          - Build Next.js standalone output on host"
	@echo "  make docker         - Build and load Docker image"
	@echo "  make docker API_BASE_URL=http://localhost:88  - LB 模式 API 地址"
	@echo "  make deploy         - Deploy to K8s (update image)"
	@echo "  make deploy-init    - Initial deployment from YAML"
	@echo "  make deploy-deps-init      - Deploy webook-fe Service (first time)"
	@echo "  make deploy-ingress        - Apply frontend ingress rules"
	@echo "  make release VERSION=v0.0.2 - Build, load and deploy"
	@echo "  make status         - Show deployment status"
	@echo "  make logs           - Show Pod logs"
	@echo "  make rollback       - Rollback to previous version"
	@echo ""
	@echo "Stop / Destroy:"
	@echo "  make stop-app       - Stop webook-fe (scale to 0)"
	@echo "  make start-app      - Start webook-fe (scale to 2)"
	@echo "  make destroy-app    - Delete deployment/service/ingress"
	@echo "  make clean-docker   - Remove local Docker image"
	@echo ""
	@echo "Current Configuration:"
	@echo "  IMAGE_NAME: $(IMAGE_NAME)"
	@echo "  VERSION: $(VERSION)"
	@echo "  API_BASE_URL (build): $(API_BASE_URL)"
	@echo "  INGRESS_HOST: $(INGRESS_HOST)"
	@echo ""
	@echo "Access:"
	@echo "  LoadBalancer: http://<EXTERNAL-IP>:89"
	@echo "  Ingress:      http://$(INGRESS_HOST)  (hosts: 127.0.0.1 $(INGRESS_HOST) api.webook.com)"

.PHONY: deploy-init
deploy-init:
	@echo "Initial deployment to new environment..."
	@kubectl apply -f $(DEPLOYMENT_FILE) -n $(NAMESPACE) || true
	@sleep 2
	@echo "Updating to version: $(VERSION)"
	@kubectl set image deployment/$(DEPLOYMENT_NAME) $(DEPLOYMENT_NAME)=$(IMAGE_NAME):$(VERSION) -n $(NAMESPACE)
	@echo "Waiting for Pods to be ready..."
	@kubectl rollout status deployment/$(DEPLOYMENT_NAME) -n $(NAMESPACE) --timeout=300s || true
	@echo "Initial deployment completed!"
	@make status

.PHONY: deploy-deps-init
deploy-deps-init:
	@echo "Applying webook-fe service..."
	@kubectl apply -f k8s-webook-fe-service.yaml -n $(NAMESPACE)
	@echo "Service applied."

.PHONY: deploy-ingress
deploy-ingress:
	@echo "Applying frontend ingress rules..."
	@kubectl apply -f k8s-ingress-fe.yaml -n $(NAMESPACE)
	@echo "Ingress applied: http://$(INGRESS_HOST) -> webook-fe:89"

.PHONY: build
build:
	@echo "Building Next.js on host..."
	@echo "  API_BASE_URL=$(API_BASE_URL)"
	@test -f node_modules/next/dist/bin/next || (echo "Missing dependencies; run: npm install" && exit 1)
	@rm -rf .next
	@NEXT_PUBLIC_API_BASE_URL=$(API_BASE_URL) npm run build
	@test -f .next/standalone/server.js || (echo "Missing .next/standalone/server.js" && exit 1)
	@echo "Build completed."

.PHONY: docker
docker: build
	@echo "Building Docker image: $(IMAGE_NAME):$(VERSION)"
	@docker rmi -f $(IMAGE_NAME):$(VERSION) 2>/dev/null || true
	@docker build -t $(IMAGE_NAME):$(VERSION) .
	@echo "Image built successfully: $(IMAGE_NAME):$(VERSION)"
	@$(MAKE) load-image

.PHONY: load-image
load-image:
	@echo "Loading image into K8s node ($(KIND_NODE))..."
	@docker inspect $(KIND_NODE) >/dev/null 2>&1 || (echo "K8s node not found: $(KIND_NODE)" && exit 1)
	@docker save $(IMAGE_NAME):$(VERSION) | docker exec -i $(KIND_NODE) ctr -n k8s.io images import -
	@echo "Image loaded: $(IMAGE_NAME):$(VERSION)"

.PHONY: deploy
deploy:
	@echo "Deploying to Kubernetes..."
	@echo "   Image: $(IMAGE_NAME):$(VERSION)"
	@if ! kubectl get deployment/$(DEPLOYMENT_NAME) -n $(NAMESPACE) >/dev/null 2>&1; then \
		echo "Deployment not found, run 'make deploy-init' first."; \
		exit 1; \
	fi
	@kubectl set image deployment/$(DEPLOYMENT_NAME) $(DEPLOYMENT_NAME)=$(IMAGE_NAME):$(VERSION) -n $(NAMESPACE)
	@kubectl rollout restart deployment/$(DEPLOYMENT_NAME) -n $(NAMESPACE)
	@kubectl rollout status deployment/$(DEPLOYMENT_NAME) -n $(NAMESPACE) --timeout=300s
	@echo "Deployment completed!"
	@$(MAKE) status

.PHONY: release
release: docker deploy
	@echo "Release completed: $(IMAGE_NAME):$(VERSION)"

.PHONY: status
status:
	@echo "Deployment:"
	@kubectl get deployment $(DEPLOYMENT_NAME) -n $(NAMESPACE) || true
	@echo ""
	@echo "Pods:"
	@kubectl get pods -l app=$(DEPLOYMENT_NAME) -n $(NAMESPACE) || true
	@echo ""
	@echo "Service:"
	@kubectl get svc $(DEPLOYMENT_NAME) -n $(NAMESPACE) || true
	@echo ""
	@echo "Ingress:"
	@kubectl get ingress webook-fe-ingress -n $(NAMESPACE) || true

.PHONY: logs
logs:
	@kubectl logs -l app=$(DEPLOYMENT_NAME) -n $(NAMESPACE) --tail=50 || true

.PHONY: logs-follow
logs-follow:
	@kubectl logs -l app=$(DEPLOYMENT_NAME) -n $(NAMESPACE) -f || true

.PHONY: rollback
rollback:
	@kubectl rollout undo deployment/$(DEPLOYMENT_NAME) -n $(NAMESPACE)
	@kubectl rollout status deployment/$(DEPLOYMENT_NAME) -n $(NAMESPACE) --timeout=300s || true
	@make status

.PHONY: version
version:
	@kubectl get deployment $(DEPLOYMENT_NAME) -n $(NAMESPACE) -o jsonpath='{.spec.template.spec.containers[0].image}' 2>/dev/null || echo "Deployment not found"
	@echo ""

.PHONY: clean-docker
clean-docker:
	@docker rmi $(IMAGE_NAME):$(VERSION) 2>/dev/null || true

.PHONY: stop-app
stop-app:
	@kubectl scale deployment/$(DEPLOYMENT_NAME) --replicas=0 -n $(NAMESPACE) 2>/dev/null || echo "Deployment not found"

.PHONY: start-app
start-app:
	@kubectl scale deployment/$(DEPLOYMENT_NAME) --replicas=2 -n $(NAMESPACE) 2>/dev/null || echo "Deployment not found"
	@kubectl rollout status deployment/$(DEPLOYMENT_NAME) -n $(NAMESPACE) --timeout=300s || true

.PHONY: destroy-app
destroy-app:
	@kubectl delete -f $(DEPLOYMENT_FILE) -n $(NAMESPACE) --ignore-not-found
	@kubectl delete -f k8s-webook-fe-service.yaml -n $(NAMESPACE) --ignore-not-found
	@kubectl delete -f k8s-ingress-fe.yaml -n $(NAMESPACE) --ignore-not-found
	@echo "webook-fe deleted."
