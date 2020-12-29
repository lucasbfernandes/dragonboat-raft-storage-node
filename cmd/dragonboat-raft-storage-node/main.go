// Copyright 2019-present Open Networking Foundation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"fmt"
	storageapi "github.com/atomix/api/go/atomix/storage"
	"github.com/atomix/dragonboat-raft-storage-node/pkg/atomix/raft"
	"github.com/atomix/dragonboat-raft-storage-node/pkg/atomix/raft/config"
	"github.com/atomix/go-framework/pkg/atomix/counter"
	"github.com/atomix/go-framework/pkg/atomix/election"
	"github.com/atomix/go-framework/pkg/atomix/indexedmap"
	"github.com/atomix/go-framework/pkg/atomix/leader"
	"github.com/atomix/go-framework/pkg/atomix/list"
	"github.com/atomix/go-framework/pkg/atomix/lock"
	logprimitive "github.com/atomix/go-framework/pkg/atomix/log"
	"github.com/atomix/go-framework/pkg/atomix/map"
	"github.com/atomix/go-framework/pkg/atomix/set"
	"github.com/atomix/go-framework/pkg/atomix/storage"
	"github.com/atomix/go-framework/pkg/atomix/value"
	"github.com/gogo/protobuf/jsonpb"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/signal"
)

func main() {
	log.SetLevel(log.InfoLevel)
	log.SetOutput(os.Stdout)

	nodeID := os.Args[1]
	storageConfig := parseStorageConfig()
	protocolConfig := parseProtocolConfig()

	// Create an Atomix node
	node := storage.NewNode(nodeID, storageConfig, raft.NewProtocol(storageConfig, protocolConfig))

	// Register primitives on the Atomix node
	counter.RegisterService(node)
	election.RegisterService(node)
	indexedmap.RegisterService(node)
	lock.RegisterService(node)
	logprimitive.RegisterService(node)
	leader.RegisterService(node)
	list.RegisterService(node)
	_map.RegisterService(node)
	set.RegisterService(node)
	value.RegisterService(node)

	// Start the node
	if err := node.Start(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Wait for an interrupt signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch

	// Stop the node after an interrupt
	if err := node.Stop(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func parseStorageConfig() storageapi.StorageConfig {
	storageConfigFile := os.Args[2]
	storageConfig := storageapi.StorageConfig{}
	nodeBytes, err := ioutil.ReadFile(storageConfigFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if err := jsonpb.Unmarshal(bytes.NewReader(nodeBytes), &storageConfig); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return storageConfig
}

func parseProtocolConfig() config.ProtocolConfig {
	protocolConfigFile := os.Args[3]
	protocolConfig := config.ProtocolConfig{}
	protocolBytes, err := ioutil.ReadFile(protocolConfigFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if len(protocolBytes) == 0 {
		return protocolConfig
	}
	if err := jsonpb.Unmarshal(bytes.NewReader(protocolBytes), &protocolConfig); err != nil {
		fmt.Println(err)
	}
	return protocolConfig
}
