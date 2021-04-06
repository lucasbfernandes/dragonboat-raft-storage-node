FROM golang:1.14-alpine

RUN apk add --no-cache git
RUN apk add build-base

WORKDIR /app/dragonboat-raft-storage-node
COPY ./go.mod .
COPY ./go.sum .

RUN go mod download

COPY . .

RUN go build -o ./out/dragonboat-raft-storage-node cmd/dragonboat-raft-storage-node/*.go

ENTRYPOINT ["./out/dragonboat-raft-storage-node"]