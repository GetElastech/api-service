
# Build dependencies
.PHONY: install-tools
install-tools:
	pushd vendor/github.com/onflow/flow-go/crypto && go generate && go build; popd

# Run API service
.PHONY: run
run:
	go run -v -tags=relic main/api-service.go

# Build API service
.PHONY: build
build:
	go build -v -tags=relic -o /app main/api-service.go

# Test API service
.PHONY: test
test:
	go test -v -tags=relic ./...

# Run API service in Docker
.PHONY: docker-run
docker-run: docker-build
	docker run -t -i --rm onflow.org/api-service go run -v -tags=relic main/api-service.go

# Run build/test/run debug console
.PHONY: debug
debug:
	docker build -t onflow.org/api-service-debug --target build-dependencies .
	docker run -t -i --rm onflow.org/api-service-debug /bin/bash

# Run all tests
.PHONY: docker-test
docker-test: docker-build-test
	docker run -t -i --rm onflow.org/api-service go test -v -tags=relic ./...

# Build production Docker container
.PHONY: docker-build
docker-build:
	docker build -t onflow.org/api-service --target production .
	docker build -t onflow.org/api-service-small --target production-small .

# Build intermediate build docker container
.PHONY: docker-build-test
docker-build-test:
	docker build -t onflow.org/api-service --target build-env .
