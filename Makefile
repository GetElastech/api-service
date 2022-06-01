
# Build dependencies
.PHONY: install-tools
install-tools:
	pushd vendor/github.com/onflow/flow-go/crypto && go generate && go build; popd

# Run API service
.PHONY: run
run:
	go run -v -tags=relic cmd/api-service/main.go

# Build API service
.PHONY: build
build:
	go build -v -tags=relic -o /app cmd/api-service/main.go

# Test API service
.PHONY: test
test:
	go test -v -tags=relic ./...

# Run API service in Docker
.PHONY: docker-run
docker-run: docker-build
	docker run -d --name flow_api_service --rm -p 4900:9000 onflow.org/api-service go run -v -tags=relic cmd/api-service/main.go

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

# Clean all
.PHONY: docker-clean
docker-clean:
	docker system prune -a
