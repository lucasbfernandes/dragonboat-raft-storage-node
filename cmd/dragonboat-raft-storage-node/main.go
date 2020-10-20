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
	"github.com/atomix/api/proto/atomix/database"
	"github.com/atomix/dragonboat-raft-storage-node/pkg/atomix/raft"
	"github.com/atomix/dragonboat-raft-storage-node/pkg/atomix/raft/config"
	"github.com/atomix/go-framework/pkg/atomix"
	"github.com/atomix/go-framework/pkg/atomix/counter"
	"github.com/atomix/go-framework/pkg/atomix/election"
	"github.com/atomix/go-framework/pkg/atomix/indexedmap"
	"github.com/atomix/go-framework/pkg/atomix/leader"
	"github.com/atomix/go-framework/pkg/atomix/list"
	"github.com/atomix/go-framework/pkg/atomix/lock"
	logprimitive "github.com/atomix/go-framework/pkg/atomix/log"
	"github.com/atomix/go-framework/pkg/atomix/map"
	"github.com/atomix/go-framework/pkg/atomix/set"
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
	clusterConfig := parseClusterConfig()
	protocolConfig := parseProtocolConfig()

	// Create an Atomix node
	node := atomix.NewNode(nodeID, clusterConfig, raft.NewProtocol(clusterConfig, protocolConfig))

	// Register primitives on the Atomix node
	counter.RegisterPrimitive(node)
	election.RegisterPrimitive(node)
	indexedmap.RegisterPrimitive(node)
	lock.RegisterPrimitive(node)
	logprimitive.RegisterPrimitive(node)
	leader.RegisterPrimitive(node)
	list.RegisterPrimitive(node)
	_map.RegisterPrimitive(node)
	set.RegisterPrimitive(node)
	value.RegisterPrimitive(node)

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

func parseClusterConfig() *database.DatabaseConfig {
	nodeConfigFile := os.Args[2]
	nodeConfig := &database.DatabaseConfig{}
	nodeBytes, err := ioutil.ReadFile(nodeConfigFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if err := jsonpb.Unmarshal(bytes.NewReader(nodeBytes), nodeConfig); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return nodeConfig
}

func parseProtocolConfig() *config.ProtocolConfig {
	protocolConfigFile := os.Args[3]
	protocolConfig := &config.ProtocolConfig{}
	protocolBytes, err := ioutil.ReadFile(protocolConfigFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if len(protocolBytes) == 0 {
		return protocolConfig
	}
	if err := jsonpb.Unmarshal(bytes.NewReader(protocolBytes), protocolConfig); err != nil {
		fmt.Println(err)
	}
	return protocolConfig
}
