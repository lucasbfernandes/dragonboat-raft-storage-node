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
	"github.com/atomix/api/proto/atomix/database"
	"github.com/atomix/dragonboat-raft-storage/pkg/atomix/raft/config"
	"github.com/atomix/go-framework/pkg/atomix/cluster"
	"github.com/atomix/go-framework/pkg/atomix/primitive"
	"github.com/lni/dragonboat/v3"
	raftconfig "github.com/lni/dragonboat/v3/config"
	"github.com/lni/dragonboat/v3/raftio"
	"github.com/lni/dragonboat/v3/statemachine"
	"sort"
	"sync"
)

const dataDir = "/var/lib/atomix/data"
const rttMillisecond = 200

// NewProtocol returns a new Raft Protocol instance
func NewProtocol(partitionConfig *database.DatabaseConfig, protocolConfig *config.ProtocolConfig) *Protocol {
	return &Protocol{
		partitionConfig: partitionConfig,
		protocolConfig:  protocolConfig,
		clients:         make(map[int]*Partition),
		servers:         make(map[int]*Server),
	}
}

// Protocol is an implementation of the Client interface providing the Raft consensus protocol
type Protocol struct {
	primitive.Protocol
	partitionConfig *database.DatabaseConfig
	protocolConfig  *config.ProtocolConfig
	mu              sync.RWMutex
	clients         map[int]*Partition
	servers         map[int]*Server
}

type startupListener struct {
	ch   chan<- int
	mu   sync.Mutex
	done bool
}

func (l *startupListener) LeaderUpdated(info raftio.LeaderInfo) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.done && info.LeaderID > 0 {
		l.ch <- int(info.ClusterID)
	}
}

func (l *startupListener) close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.done = true
}

// Start starts the Raft protocol
func (p *Protocol) Start(clusterConfig cluster.Cluster, registry primitive.Registry) error {
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

	// Create a listener to wait for a leader to be elected
	startupCh := make(chan int)
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
		p.clients[int(clusterID)] = newClient(clusterID, nodeID, node, clientMembers, streams)
		p.mu.Unlock()
		return fsm
	}

	for _, partition := range p.partitionConfig.Partitions {
		config := raftconfig.Config{
			NodeID:             nodeID,
			ClusterID:          uint64(partition.Partition),
			ElectionRTT:        10,
			HeartbeatRTT:       1,
			CheckQuorum:        true,
			SnapshotEntries:    p.protocolConfig.GetSnapshotThresholdOrDefault(),
			CompactionOverhead: p.protocolConfig.GetSnapshotThresholdOrDefault() / 10,
		}

		server := newServer(uint64(partition.Partition), serverMembers, node, config, fsmFactory)
		if err := server.Start(); err != nil {
			return err
		}
		p.servers[int(partition.Partition)] = server
	}

	started := make(map[int]bool)
	for partitionID := range startupCh {
		started[partitionID] = true
		if len(started) == len(p.servers) {
			listener.close()
			return nil
		}
	}
	return nil
}

// Partition returns the given partition client
func (p *Protocol) Partition(partitionID int) primitive.Partition {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.clients[partitionID]
}

// Partitions returns all partition clients
func (p *Protocol) Partitions() []primitive.Partition {
	p.mu.RLock()
	defer p.mu.RUnlock()
	partitions := make([]primitive.Partition, 0, len(p.clients))
	for _, client := range p.clients {
		partitions = append(partitions, client)
	}
	sort.Slice(partitions, func(i, j int) bool {
		return partitions[i].(*Partition).clusterID < partitions[j].(*Partition).clusterID
	})
	return partitions
}

// Stop stops the Raft protocol
func (p *Protocol) Stop() error {
	var returnErr error
	for _, server := range p.servers {
		if err := server.Stop(); err != nil {
			returnErr = err
		}
	}
	return returnErr
}
