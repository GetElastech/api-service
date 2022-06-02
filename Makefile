
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

# Run API service attached to localnet in Docker
.PHONY: docker-run-localnet
docker-run-localnet: docker-build
	docker run -d --name localnet_flow_api_service --rm -p 127.0.0.1:9500:9000 --network localnet_default \
		--link access_1:access onflow.org/api-service go run -v -tags=relic cmd/api-service/main.go \
		--upstream-node-addresses=access:9000 --upstream-node-public-keys=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa --rpc-addr=:9000
	sleep 10
	# To follow: docker logs -f localnet_flow_api_service
	docker logs localnet_flow_api_service
	# Check latest block: flow -f ./flow-localnet.json -n api blocks get latest
	docker run -t -i --rm --link localnet_flow_api_service:flow_api  --network localnet_default \
		onflow.org/flow-e2e-test
	docker stop localnet_flow_api_service

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
	docker build -t onflow.org/flow-cli --target flow-cli .
	docker build -t onflow.org/flow-e2e-test --target flow-e2e-test .

# Build intermediate build docker container
.PHONY: docker-build-test
docker-build-test:
	docker build -t onflow.org/api-service --target build-env .

# Clean all
.PHONY: docker-clean
docker-clean:
	docker system prune -a
