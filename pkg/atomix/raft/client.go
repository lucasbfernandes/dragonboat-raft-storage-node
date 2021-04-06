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
	"context"
	streams "github.com/lucasbfernandes/go-framework/pkg/atomix/stream"
	"github.com/gogo/protobuf/proto"
	"github.com/lni/dragonboat/v3"
	"time"
)

const clientTimeout = 15 * time.Second

// newClient returns a new Raft consensus protocol client
func newClient(clusterID uint64, nodeID uint64, node *dragonboat.NodeHost, members map[uint64]string, streams *streamManager) *Partition {
	return &Partition{
		clusterID: clusterID,
		nodeID:    nodeID,
		node:      node,
		members:   members,
		streams:   streams,
	}
}

// Client is the Raft client
type Partition struct {
	clusterID uint64
	nodeID    uint64
	node      *dragonboat.NodeHost
	members   map[uint64]string
	streams   *streamManager
}

func (c *Partition) MustLeader() bool {
	return true
}

func (c *Partition) IsLeader() bool {
	leader, ok, err := c.node.GetLeaderID(c.clusterID)
	if !ok || err != nil {
		return false
	}
	return leader == c.nodeID
}

func (c *Partition) Leader() string {
	leader, ok, err := c.node.GetLeaderID(c.clusterID)
	if !ok || err != nil {
		return ""
	}
	return c.members[leader]
}

func (c *Partition) Write(ctx context.Context, input []byte, stream streams.WriteStream) error {
	streamID, stream := c.streams.addStream(stream)
	entry := &Entry{
		Value:     input,
		StreamID:  streamID,
		Timestamp: time.Now(),
	}
	bytes, err := proto.Marshal(entry)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), clientTimeout)
	defer cancel()
	if _, err := c.node.SyncPropose(ctx, c.node.GetNoOPSession(c.clusterID), bytes); err != nil {
		stream.Close()
		return err
	}
	return nil
}

func (c *Partition) Read(ctx context.Context, input []byte, stream streams.WriteStream) error {
	query := queryContext{
		value:  input,
		stream: stream,
	}
	ctx, cancel := context.WithTimeout(context.Background(), clientTimeout)
	defer cancel()
	if _, err := c.node.SyncRead(ctx, c.clusterID, query); err != nil {
		stream.Close()
		return err
	}
	return nil
}
