export CGO_ENABLED=0
export GO111MODULE=on

.PHONY: build

ATOMIX_DRAGONBOAT_RAFT_NODE_VERSION := latest

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
	CGO_ENABLED=1 $(GOCMD) -o build/_output/dragonboat-raft-replica ./cmd/dragonboat-raft-replica

test: # @HELP run the unit tests and source code validation
test: build license_check linters
	go test github.com/atomix/dragonboat-raft-replica/...

coverage: # @HELP generate unit test coverage data
coverage: build linters license_check
	go test github.com/atomix/dragonboat-raft-replica/pkg/... -coverprofile=coverage.out.tmp -covermode=count
	@cat coverage.out.tmp | grep -v ".pb.go" > coverage.out

linters: # @HELP examines Go source code and reports coding problems
	golangci-lint run

license_check: # @HELP examine and ensure license headers exist
	./build/licensing/boilerplate.py -v

proto: # @HELP build Protobuf/gRPC generated types
proto:
	docker run -it -v `pwd`:/go/src/github.com/atomix/dragonboat-raft-replica \
		-w /go/src/github.com/atomix/dragonboat-raft-replica \
		--entrypoint build/bin/compile_protos.sh \
		onosproject/protoc-go:stable

image: # @HELP build dragonboat-raft-replica Docker image
	docker build . -f build/docker/Dockerfile -t atomix/dragonboat-raft-replica:${ATOMIX_DRAGONBOAT_RAFT_NODE_VERSION}

push: # @HELP push dragonboat-raft-replica Docker image
	docker push atomix/dragonboat-raft-replica:${ATOMIX_DRAGONBOAT_RAFT_NODE_VERSION}
