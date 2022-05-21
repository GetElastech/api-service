# NOTE: Must be run in the context of the repo's root directory

## (1) Download a suitable version of Go
# FIX: We use 1.17 due to unsupported modules
# FIX: github.com/lucas-clemente/quic-go@v0.24.0/internal/qtls/go118.go:6:13:
# FIX:   cannot use "quic-go doesn't build on Go 1.18 yet."
FROM golang:1.17 AS build-setup

# Add optional items like apt install -y make cmake gcc g++
RUN apt update && apt install -y cmake make gcc g++

## (2) Build the app binary
FROM build-setup AS build-dependencies

# Cache gopath dependencies for faster builds
# Newer projects should opt for go mod vendor for reliability and security
RUN mkdir /app
RUN mkdir /app/src
COPY src/go.mod /app/src
COPY src/go.sum /app/src

# FIX: This generates code marked by `go:build relic` and `+build relic`. See `combined_verifier_v3.go`.
# FIX: This is not needed, if vendor/ is used
WORKDIR /app/src
RUN go mod download
RUN go mod download github.com/onflow/flow-go/crypto@v0.24.3
RUN cd $GOPATH/pkg/mod/github.com/onflow/flow-go/crypto@v0.24.3 && go generate && go build

# FIX: Devs should review all what they use to limit build time
RUN cat go.sum

## (3) Build the app binary
FROM build-dependencies AS build-env

COPY src /app/src
WORKDIR /app/src

# Fix: make sure no further steps update modules later, so that we can debug regressions
RUN go mod vendor
RUN cp -R $GOPATH/pkg/mod/github.com/onflow/flow-go/crypto@v0.24.3/* /app/src/vendor/github.com/onflow/flow-go/crypto
RUN ls /app/src/vendor/github.com/onflow/flow-go/crypto/relic

# FIX: Without -tags=relic we get undefined: "github.com/onflow/flow-go/consensus/hotstuff/verification".NewCombinedVerifier
RUN go build -v -tags=relic -o /app main/api-service.go
RUN cp /app/api-service /app/application

CMD /bin/bash

## (5) Add the statically linked binary to a distroless image
FROM build-env as production

WORKDIR /app/src
COPY --from=build-env /app/api-service /app/api-service

#RUN touch bootstrap/private-root-information/private-node-info_a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5/node-info.priv.json

CMD ["go", "run", "-tags=relic", "main/api-service.go",  \
    "--secretsdir=/data/secrets"]
#    "--nodeid=a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5"]

## (6) Add the statically linked binary to a distroless image
FROM golang:1.17 as production-small

RUN rm -rf /go
RUN rm -rf /app
COPY --from=production /app/api-service /bin/api-service

CMD ["/bin/api-service"]
