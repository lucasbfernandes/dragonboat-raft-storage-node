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

COPY go.mod /go/src/github.com/atomix/dragonboat-raft-replica/

RUN cd /go/src/github.com/atomix/dragonboat-raft-replica && \
    go mod download && \
    rm -rf /go/src/github.com/atomix/dragonboat-raft-replica

COPY . /go/src/github.com/atomix/dragonboat-raft-replica/

RUN cd /go/src/github.com/atomix/dragonboat-raft-replica && \
    make && \
    mv /go/src/github.com/atomix/dragonboat-raft-replica/build/_output/dragonboat-raft-replica /usr/local/bin/dragonboat-raft-replica

ENTRYPOINT ["dragonboat-raft-replica"]
