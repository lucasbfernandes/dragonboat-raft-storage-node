FROM ubuntu:18.04 as build

RUN apt-get update && apt-get install -y --no-install-recommends \
  ca-certificates \
  g++ \
  gcc \
  git \
  libc6-dev \
  make \
  wget \
  librocksdb-dev \
  && rm -rf /var/lib/apt/lists/*

RUN set -eux; \
  url="https://dl.google.com/go/go1.12.6.linux-amd64.tar.gz"; \
  wget -O go.tgz "$url"; \
  tar -C /usr/local -xzf go.tgz; \
  rm go.tgz; \
  export PATH="/usr/local/go/bin:$PATH"; \
  go version

ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH
RUN go get -u google.golang.org/grpc

ENV GO111MODULE=on

COPY Makefile go.mod go.sum /go/src/github.com/atomix/dragonboat-raft-storage/
COPY vendor/ /go/src/github.com/atomix/dragonboat-raft-storage/vendor/
COPY pkg/ /go/src/github.com/atomix/dragonboat-raft-storage/pkg/
COPY cmd/ /go/src/github.com/atomix/dragonboat-raft-storage/cmd/

RUN cd /go/src/github.com/atomix/dragonboat-raft-storage && make

FROM ubuntu:18.04

RUN apt-get update && apt-get install -y --no-install-recommends \
  librocksdb-dev \
  && rm -rf /var/lib/apt/lists/*

COPY --from=build /go/src/github.com/atomix/dragonboat-raft-storage/build/_output/raft-storage-node /usr/local/bin/raft-storage-node

ENTRYPOINT ["raft-storage-node"]
