.PHONY: build test

REGISTRY=semaphoreci/metrics-apiserver
LATEST_VERSION=$(shell git tag | sort --version-sort | tail -n 1)

SECURITY_TOOLBOX_BRANCH ?= master
SECURITY_TOOLBOX_TMP_DIR ?= /tmp/security-toolbox

check.prepare:
	rm -rf $(SECURITY_TOOLBOX_TMP_DIR)
	git clone git@github.com:renderedtext/security-toolbox.git $(SECURITY_TOOLBOX_TMP_DIR) && (cd $(SECURITY_TOOLBOX_TMP_DIR) && git checkout $(SECURITY_TOOLBOX_BRANCH) && cd -)

check.static: check.prepare
	docker run -it -v $$(pwd):/app \
		-v $(SECURITY_TOOLBOX_TMP_DIR):$(SECURITY_TOOLBOX_TMP_DIR) \
		registry.semaphoreci.com/ruby:2.7 \
		bash -c 'cd /app && $(SECURITY_TOOLBOX_TMP_DIR)/code --language go -d'

check.deps: check.prepare
	docker run -it -v $$(pwd):/app \
		-v $(SECURITY_TOOLBOX_TMP_DIR):$(SECURITY_TOOLBOX_TMP_DIR) \
		registry.semaphoreci.com/ruby:2.7 \
		bash -c 'cd /app && $(SECURITY_TOOLBOX_TMP_DIR)/dependencies --language go -d'

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
