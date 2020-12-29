export CGO_ENABLED=0
export GO111MODULE=on

.PHONY: build

ifdef VERSION
RAFT_STORAGE_NODE_VERSION := $(VERSION)
else
RAFT_STORAGE_NODE_VERSION := latest
endif

all: build

build: # @HELP build the source code
build:
	GOOS=linux GOARCH=amd64 go build -o build/_output/dragonboat-raft-storage-node ./cmd/dragonboat-raft-storage-node
	GOOS=linux GOARCH=amd64 go build -o build/_output/dragonboat-raft-storage-proxy ./cmd/dragonboat-raft-storage-proxy

test: # @HELP run the unit tests and source code validation
test: build license_check linters
	go test github.com/atomix/dragonboat-raft-storage-node/...

coverage: # @HELP generate unit test coverage data
coverage: build linters license_check
	go test github.com/atomix/dragonboat-raft-storage-node/pkg/... -coverprofile=coverage.out.tmp -covermode=count
	@cat coverage.out.tmp | grep -v ".pb.go" > coverage.out

linters: # @HELP examines Go source code and reports coding problems
	golangci-lint run

license_check: # @HELP examine and ensure license headers exist
	./build/licensing/boilerplate.py -v

proto: # @HELP build Protobuf/gRPC generated types
proto:
	docker run -it -v `pwd`:/go/src/github.com/atomix/dragonboat-raft-storage-node \
		-w /go/src/github.com/atomix/dragonboat-raft-storage-node \
		--entrypoint build/bin/compile_protos.sh \
		onosproject/protoc-go:stable

image: # @HELP build etcd-raft-replica Docker image
image: build
	docker build . -f build/dragonboat-raft-storage-node/Dockerfile -t atomix/dragonboat-raft-storage-node:${RAFT_STORAGE_NODE_VERSION}
	docker build . -f build/dragonboat-raft-storage-proxy/Dockerfile -t atomix/dragonboat-raft-storage-proxy:${RAFT_STORAGE_NODE_VERSION}

clean: # @HELP clean build files
	@rm -rf vendor build/_output

push: # @HELP push dragonboat-raft-storage-node Docker image
	docker push atomix/dragonboat-raft-storage-node:${RAFT_STORAGE_NODE_VERSION}
	docker push atomix/dragonboat-raft-storage-proxy:${RAFT_STORAGE_NODE_VERSION}
