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
	"github.com/atomix/api/go/atomix/protocol"
	raft "github.com/atomix/dragonboat-raft-storage-node/pkg/storage"
	"github.com/atomix/dragonboat-raft-storage-node/pkg/storage/config"
	"github.com/atomix/go-framework/pkg/atomix/cluster"
	"github.com/atomix/go-framework/pkg/atomix/protocol/rsm"
	"github.com/atomix/go-framework/pkg/atomix/protocol/rsm/counter"
	"github.com/atomix/go-framework/pkg/atomix/protocol/rsm/election"
	"github.com/atomix/go-framework/pkg/atomix/protocol/rsm/indexedmap"
	"github.com/atomix/go-framework/pkg/atomix/protocol/rsm/leader"
	"github.com/atomix/go-framework/pkg/atomix/protocol/rsm/list"
	"github.com/atomix/go-framework/pkg/atomix/protocol/rsm/lock"
	logprimitive "github.com/atomix/go-framework/pkg/atomix/protocol/rsm/log"
	"github.com/atomix/go-framework/pkg/atomix/protocol/rsm/map"
	"github.com/atomix/go-framework/pkg/atomix/protocol/rsm/set"
	"github.com/atomix/go-framework/pkg/atomix/protocol/rsm/value"
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
	protocolConfig := parseProtocolConfig()
	raftConfig := parseRaftConfig()

	cluster := cluster.NewCluster(protocolConfig, cluster.WithMemberID(nodeID))

	// Create an Atomix node
	node := rsm.NewNode(cluster, raft.NewProtocol(raftConfig))

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

func parseProtocolConfig() protocol.ProtocolConfig {
	configFile := os.Args[2]
	config := protocol.ProtocolConfig{}
	nodeBytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if err := jsonpb.Unmarshal(bytes.NewReader(nodeBytes), &config); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return config
}

func parseRaftConfig() config.ProtocolConfig {
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
