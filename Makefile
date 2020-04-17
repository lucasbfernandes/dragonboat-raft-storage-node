export CGO_ENABLED=0
export GO111MODULE=on

.PHONY: build

ifdef VERSION
RAFT_STORAGE_NODE_VERSION := $(VERSION)
else
RAFT_STORAGE_NODE_VERSION := latest
endif

OS := $(shell uname)
ROCKSDB_MAJOR_VER=5
ifeq ($(OS),Darwin)
ROCKSDB_SO_FILE=librocksdb.$(ROCKSDB_MAJOR_VER).dylib
else ifeq ($(OS),Linux)
ROCKSDB_SO_FILE=librocksdb.so.$(ROCKSDB_MAJOR_VER)
else
$(error OS type $(OS) not supported)
endif

ROCKSDB_INC_PATH ?=
ROCKSDB_LIB_PATH ?=
# in /usr/local/lib?
ifeq ($(ROCKSDB_LIB_PATH),)
ifeq ($(ROCKSDB_INC_PATH),)
ifneq ($(wildcard /usr/local/lib/$(ROCKSDB_SO_FILE)),)
ifneq ($(wildcard /usr/local/include/rocksdb/c.h),)
$(info rocksdb lib found at /usr/local/lib/$(ROCKSDB_SO_FILE))
ROCKSDB_LIB_PATH=/usr/local/lib
endif
endif
endif
endif

ifeq ($(ROCKSDB_LIB_PATH),)
CDEPS_LDFLAGS=-lrocksdb
else
CDEPS_LDFLAGS=-L$(ROCKSDB_LIB_PATH) -lrocksdb
endif
ifneq ($(ROCKSDB_INC_PATH),)
CGO_CXXFLAGS=CGO_CFLAGS="-I$(ROCKSDB_INC_PATH)"
endif
CGO_LDFLAGS=CGO_LDFLAGS="$(CDEPS_LDFLAGS)"
GOCMD=$(CGO_LDFLAGS) $(CGO_CXXFLAGS) go build -v

all: build

build: # @HELP build the source code
build:
	CGO_ENABLED=1 $(GOCMD) -mod vendor -o build/_output/raft-storage-node ./cmd/raft-storage-node

test: # @HELP run the unit tests and source code validation
test: build license_check linters
	go test github.com/atomix/raft-storage-node/...


coverage: # @HELP generate unit test coverage data
coverage: build linters license_check
	go test github.com/atomix/dragonboat-raft-storage/pkg/... -coverprofile=coverage.out.tmp -covermode=count
	@cat coverage.out.tmp | grep -v ".pb.go" > coverage.out

linters: # @HELP examines Go source code and reports coding problems
	golangci-lint run

license_check: # @HELP examine and ensure license headers exist
	./build/licensing/boilerplate.py -v

proto: # @HELP build Protobuf/gRPC generated types
proto:
	docker run -it -v `pwd`:/go/src/github.com/atomix/raft-storage-node \
		-w /go/src/github.com/atomix/raft-storage-node \
		--entrypoint build/bin/compile_protos.sh \
		onosproject/protoc-go:stable

images: # @HELP build raft-storage-node Docker image
	@go mod vendor
	docker build . -f build/raft-storage-node/Dockerfile -t atomix/raft-storage-node:${RAFT_STORAGE_NODE_VERSION}
	@rm -r vendor

clean: # @HELP clean build files
	@rm -rf vendor build/_output

push: # @HELP push raft-storage-node Docker image
	docker push atomix/raft-storage-node:${RAFT_STORAGE_NODE_VERSION}
