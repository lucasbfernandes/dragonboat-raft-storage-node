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
	"encoding/binary"
	"github.com/atomix/go-framework/pkg/atomix/cluster"
	"github.com/atomix/go-framework/pkg/atomix/node"
	"github.com/atomix/go-framework/pkg/atomix/service"
	"github.com/atomix/go-framework/pkg/atomix/stream"
	"github.com/gogo/protobuf/proto"
	"github.com/lni/dragonboat/v3/statemachine"
	"io"
	"sync"
	"time"
)

// newStateMachine returns a new primitive state machine
func newStateMachine(cluster cluster.Cluster, registry *node.Registry, streams *streamManager) *StateMachine {
	fsm := &StateMachine{
		node:    cluster.MemberID,
		streams: streams,
	}
	fsm.state = service.NewManager(registry, fsm)
	return fsm
}

type StateMachine struct {
	node      string
	state     node.StateMachine
	streams   *streamManager
	index     uint64
	timestamp time.Time
	mu        sync.Mutex
}

func (s *StateMachine) Node() string {
	return s.node
}

func (s *StateMachine) Index() uint64 {
	return s.index
}

func (s *StateMachine) Timestamp() time.Time {
	return s.timestamp
}

func (s *StateMachine) Update(bytes []byte) (statemachine.Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tsEntry := &Entry{}
	if err := proto.Unmarshal(bytes, tsEntry); err != nil {
		return statemachine.Result{}, err
	}

	stream := s.streams.getStream(tsEntry.StreamID)

	s.index++
	if tsEntry.Timestamp.After(s.timestamp) {
		s.timestamp = tsEntry.Timestamp
	}
	s.state.Command(tsEntry.Value, stream)
	return statemachine.Result{}, nil
}

func (s *StateMachine) Lookup(value interface{}) (interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	query := value.(queryContext)
	s.state.Query(query.value, query.stream)
	return nil, nil
}

func (s *StateMachine) SaveSnapshot(writer io.Writer, files statemachine.ISnapshotFileCollection, done <-chan struct{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bytes, s.index)
	if _, err := writer.Write(bytes); err != nil {
		return err
	}
	binary.BigEndian.PutUint64(bytes, uint64(s.timestamp.Second()))
	if _, err := writer.Write(bytes); err != nil {
		return err
	}
	binary.BigEndian.PutUint64(bytes, uint64(s.timestamp.Nanosecond()))
	if _, err := writer.Write(bytes); err != nil {
		return err
	}
	return s.state.Snapshot(writer)
}

func (s *StateMachine) RecoverFromSnapshot(reader io.Reader, files []statemachine.SnapshotFile, done <-chan struct{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	bytes := make([]byte, 8)
	if _, err := reader.Read(bytes); err != nil {
		return err
	}
	s.index = binary.BigEndian.Uint64(bytes)
	if _, err := reader.Read(bytes); err != nil {
		return err
	}
	secs := int64(binary.BigEndian.Uint64(bytes))
	if _, err := reader.Read(bytes); err != nil {
		return err
	}
	nanos := int64(binary.BigEndian.Uint64(bytes))
	s.timestamp = time.Unix(secs, nanos)
	return s.state.Install(reader)
}

func (s *StateMachine) Close() error {
	return nil
}

type queryContext struct {
	value  []byte
	stream stream.WriteStream
}
