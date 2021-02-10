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

package storage

import (
	"fmt"
	"github.com/atomix/dragonboat-raft-storage-node/pkg/storage/config"
	"github.com/atomix/go-framework/pkg/atomix/cluster"
	"github.com/atomix/go-framework/pkg/atomix/errors"
	"github.com/atomix/go-framework/pkg/atomix/protocol/rsm"
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
func NewProtocol(config config.ProtocolConfig) *Protocol {
	return &Protocol{
		config:  config,
		clients: make(map[rsm.PartitionID]*Partition),
		servers: make(map[rsm.PartitionID]*Server),
	}
}

// Protocol is an implementation of the Client interface providing the Raft consensus protocol
type Protocol struct {
	rsm.Protocol
	config  config.ProtocolConfig
	mu      sync.RWMutex
	clients map[rsm.PartitionID]*Partition
	servers map[rsm.PartitionID]*Server
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
	close(l.ch)
}

// Start starts the Raft protocol
func (p *Protocol) Start(c *cluster.Cluster, registry rsm.Registry) error {
	member, ok := c.Member()
	if !ok {
		return errors.NewInternal("local member not configured")
	}

	address := fmt.Sprintf("%s:%d", member.Host, member.Port)

	replicas := make([]*cluster.Replica, 0, len(c.Replicas()))
	for _, replica := range c.Replicas() {
		replicas = append(replicas, replica)
	}
	sort.Slice(replicas, func(i, j int) bool {
		return replicas[i].ID < replicas[j].ID
	})

	var nodeID uint64
	clientMembers := make(map[uint64]string)
	serverMembers := make(map[uint64]string)
	for i, replica := range replicas {
		clientMembers[uint64(i+1)] = fmt.Sprintf("%s:%d", replica.Host, replica.Port)
		serverMembers[uint64(i+1)] = fmt.Sprintf("%s:8080", replica.Host)
		if replica.ID == member.ID {
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
		fsm := newStateMachine(c, rsm.PartitionID(clusterID), registry, streams)
		p.mu.Lock()
		p.clients[rsm.PartitionID(clusterID)] = newClient(clusterID, nodeID, node, clientMembers, streams)
		p.mu.Unlock()
		return fsm
	}

	for _, partition := range c.Partitions() {
		config := raftconfig.Config{
			NodeID:             nodeID,
			ClusterID:          uint64(partition.ID),
			ElectionRTT:        10,
			HeartbeatRTT:       1,
			CheckQuorum:        true,
			SnapshotEntries:    p.config.GetSnapshotThresholdOrDefault(),
			CompactionOverhead: p.config.GetSnapshotThresholdOrDefault() / 10,
		}

		server := newServer(uint64(partition.ID), serverMembers, node, config, fsmFactory)
		if err := server.Start(); err != nil {
			return err
		}
		p.servers[rsm.PartitionID(partition.ID)] = server
	}

	startedCh := make(chan struct{})
	go func() {
		startedPartitions := make(map[int]bool)
		started := false
		for partitionID := range startupCh {
			startedPartitions[partitionID] = true
			if !started && len(startedPartitions) == len(p.servers) {
				go listener.close()
				close(startedCh)
				started = true
			}
		}
	}()
	<-startedCh
	return nil
}

// Partition returns the given partition client
func (p *Protocol) Partition(partitionID rsm.PartitionID) rsm.Partition {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.clients[partitionID]
}

// Partitions returns all partition clients
func (p *Protocol) Partitions() []rsm.Partition {
	p.mu.RLock()
	defer p.mu.RUnlock()
	partitions := make([]rsm.Partition, 0, len(p.clients))
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
