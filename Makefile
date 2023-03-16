.PHONY: build test

REGISTRY=semaphoreci/metrics-apiserver
LATEST_VERSION=$(shell git tag | sort --version-sort | tail -n 1)

test.setup:
	docker compose build
	docker compose run app go get ./...

test:
	docker compose run --rm app gotestsum --format short-verbose --junitfile junit-report.xml --packages="./..." -- -p 1

build:
	rm -rf build
	env GOOS=linux GOARCH=386 go build -o build/adapter main.go

docker.build: build
	docker build -t $(REGISTRY):latest .

docker.push:
	@if [ -z "$(LATEST_VERSION)" ]; then \
		docker push $(REGISTRY):latest; \
	else \
		docker tag $(REGISTRY):latest $(REGISTRY):$(LATEST_VERSION); \
		docker push $(REGISTRY):$(LATEST_VERSION); \
		docker push $(REGISTRY):latest; \
	fi
