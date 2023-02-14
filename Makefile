build:
	rm -rf build
	env GOOS=linux GOARCH=386 go build -o build/adapter main.go

build.image:
	docker build -t semaphore-metrics-exporter .

build.agent.image:
	docker build -t semaphore-agent-k8s -f Dockerfile.agent .

k8s.metrics.create:
	kubectl apply -f k8s/metrics/rbac.yml
	kubectl apply -f k8s/metrics/service.yml
	kubectl apply -f k8s/metrics/apiservice.yml
	kubectl apply -f k8s/metrics/deployment.yml

k8s.metrics.delete:
	kubectl delete -f k8s/metrics/deployment.yml
	kubectl delete -f k8s/metrics/apiservice.yml
	kubectl delete -f k8s/metrics/service.yml
	kubectl delete -f k8s/metrics/rbac.yml

k8s.agent.create:
	kubectl apply -f k8s/agent/rbac.yml
	kubectl apply -f k8s/agent/config-map.yml
	kubectl apply -f k8s/agent/deployment.yml
	kubectl apply -f k8s/agent/scale-up-policy.yml
	kubectl apply -f k8s/agent/scale-down-policy.yml

k8s.agent.delete:
	kubectl delete -f k8s/agent/scale-down-policy.yml
	kubectl delete -f k8s/agent/scale-up-policy.yml
	kubectl delete -f k8s/agent/deployment.yml
	kubectl delete -f k8s/agent/config-map.yml
	kubectl delete -f k8s/agent/rbac.yml