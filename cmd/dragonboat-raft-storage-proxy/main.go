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
	"github.com/atomix/go-framework/pkg/atomix/cluster"
	"github.com/atomix/go-framework/pkg/atomix/proxy/rsm"
	"github.com/atomix/go-framework/pkg/atomix/proxy/rsm/counter"
	"github.com/atomix/go-framework/pkg/atomix/proxy/rsm/election"
	"github.com/atomix/go-framework/pkg/atomix/proxy/rsm/indexedmap"
	"github.com/atomix/go-framework/pkg/atomix/proxy/rsm/leader"
	"github.com/atomix/go-framework/pkg/atomix/proxy/rsm/list"
	"github.com/atomix/go-framework/pkg/atomix/proxy/rsm/lock"
	logprimitive "github.com/atomix/go-framework/pkg/atomix/proxy/rsm/log"
	"github.com/atomix/go-framework/pkg/atomix/proxy/rsm/map"
	"github.com/atomix/go-framework/pkg/atomix/proxy/rsm/set"
	"github.com/atomix/go-framework/pkg/atomix/proxy/rsm/value"
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
	config := parseConfig()

	// Create an Atomix node
	cluster := cluster.NewCluster(config, cluster.WithMemberID(nodeID))
	node := rsm.NewNode(cluster)

	// Register primitives on the Atomix node
	counter.RegisterProxy(node)
	election.RegisterProxy(node)
	indexedmap.RegisterProxy(node)
	lock.RegisterProxy(node)
	logprimitive.RegisterProxy(node)
	leader.RegisterProxy(node)
	list.RegisterProxy(node)
	_map.RegisterProxy(node)
	set.RegisterProxy(node)
	value.RegisterProxy(node)

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

func parseConfig() protocol.ProtocolConfig {
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
