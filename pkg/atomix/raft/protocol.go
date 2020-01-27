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

package raft

import (
	"fmt"
	"github.com/atomix/api/proto/atomix/controller"
	"github.com/atomix/dragonboat-raft-replica/pkg/atomix/raft/config"
	"github.com/atomix/go-framework/pkg/atomix/cluster"
	"github.com/atomix/go-framework/pkg/atomix/node"
	"github.com/lni/dragonboat/v3"
	raftconfig "github.com/lni/dragonboat/v3/config"
	"github.com/lni/dragonboat/v3/raftio"
	"github.com/lni/dragonboat/v3/statemachine"
	"hash/fnv"
	"sort"
	"sync"
)

const dataDir = "/var/lib/atomix/data"
const rttMillisecond = 200

// NewProtocol returns a new Raft Protocol instance
func NewProtocol(partitionConfig *controller.PartitionConfig, protocolConfig *config.ProtocolConfig) *Protocol {
	return &Protocol{
		partitionConfig: partitionConfig,
		protocolConfig:  protocolConfig,
	}
}

// Protocol is an implementation of the Client interface providing the Raft consensus protocol
type Protocol struct {
	node.Protocol
	partitionConfig *controller.PartitionConfig
	protocolConfig  *config.ProtocolConfig
	mu              sync.RWMutex
	client          *Client
	server          *Server
}

type startupListener struct {
	ch   chan<- struct{}
	mu   sync.Mutex
	done bool
}

func (l *startupListener) LeaderUpdated(info raftio.LeaderInfo) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.done && info.LeaderID > 0 {
		close(l.ch)
		l.done = true
	}
}

// Start starts the Raft protocol
func (p *Protocol) Start(clusterConfig cluster.Cluster, registry *node.Registry) error {
	member := clusterConfig.Members[clusterConfig.MemberID]
	address := fmt.Sprintf("%s:%d", member.Host, member.ProtocolPort)

	nodes := make([]cluster.Member, 0, len(clusterConfig.Members))
	for _, member := range clusterConfig.Members {
		nodes = append(nodes, member)
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ID < nodes[j].ID
	})

	var nodeID uint64
	clientMembers := make(map[uint64]string)
	serverMembers := make(map[uint64]string)
	for i, member := range nodes {
		clientMembers[uint64(i+1)] = fmt.Sprintf("%s:%d", member.Host, member.APIPort)
		serverMembers[uint64(i+1)] = fmt.Sprintf("%s:%d", member.Host, member.ProtocolPort)
		if member.ID == clusterConfig.MemberID {
			nodeID = uint64(i + 1)
		}
	}

	// Compute a cluster ID from the partition ID
	partitionID := fmt.Sprintf("%s-%s-%d", p.partitionConfig.Partition.Group.Namespace, p.partitionConfig.Partition.Group.Name, p.partitionConfig.Partition.Partition)
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(partitionID))
	clusterID := hash.Sum64()

	config := raftconfig.Config{
		NodeID:             nodeID,
		ClusterID:          clusterID,
		ElectionRTT:        10,
		HeartbeatRTT:       1,
		CheckQuorum:        true,
		SnapshotEntries:    p.protocolConfig.GetSnapshotThresholdOrDefault(),
		CompactionOverhead: p.protocolConfig.GetSnapshotThresholdOrDefault() / 10,
	}

	// Create a listener to wait for a leader to be elected
	startupCh := make(chan struct{})
	listener := &startupListener{
		ch: startupCh,
	}

	nodeConfig := raftconfig.NodeHostConfig{
		WALDir:            dataDir,
		NodeHostDir:       dataDir,
		RTTMillisecond:    rttMillisecond,
		RaftAddress:       address,
		RaftEventListener: listener,
	}

	node, err := dragonboat.NewNodeHost(nodeConfig)
	if err != nil {
		return err
	}

	fsmFactory := func(clusterID, nodeID uint64) statemachine.IStateMachine {
		streams := newStreamManager()
		fsm := newStateMachine(clusterConfig, registry, streams)
		p.mu.Lock()
		p.client = newClient(clusterID, nodeID, node, clientMembers, streams)
		p.mu.Unlock()
		return fsm
	}

	p.server = newServer(clusterID, serverMembers, node, config, fsmFactory)
	if err := p.server.Start(); err != nil {
		return err
	}
	<-startupCh
	return nil
}

// Client returns the Raft protocol client
func (p *Protocol) Client() node.Client {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.client
}

// Stop stops the Raft protocol
func (p *Protocol) Stop() error {
	return p.server.Stop()
}
