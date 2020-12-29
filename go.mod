module github.com/atomix/dragonboat-raft-storage-node

go 1.12

require (
	github.com/atomix/api/go v0.3.3
	github.com/atomix/go-framework v0.5.1
	github.com/gogo/protobuf v1.3.1
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/lni/dragonboat/v3 v3.1.1-0.20201211124920-79d5e54396f7
	github.com/sirupsen/logrus v1.4.2
	github.com/stretchr/testify v1.6.1
	google.golang.org/grpc v1.33.2
)

replace github.com/atomix/api/go => ../atomix-api/go
replace github.com/atomix/go-framework => ../atomix-go-node