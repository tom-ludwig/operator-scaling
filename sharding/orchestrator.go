/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sharding

import (
	"encoding/binary"
	"sync"

	"github.com/buraksezer/consistent"
	"github.com/zeebo/blake3"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// member implements consistent.Member interface.
type member string

func (m member) String() string { return string(m) }

// blake3Hasher implements consistent.Hasher using Blake3.
type blake3Hasher struct{}

func (h blake3Hasher) Sum64(data []byte) uint64 {
	sum := blake3.Sum256(data)
	return binary.LittleEndian.Uint64(sum[:8])
}

// ShardOrchestrator manages the consistent hash ring for resource sharding.
// It determines which pod should handle which resources based on consistent hashing.
type ShardOrchestrator struct {
	ring          *consistent.Consistent
	mu            sync.RWMutex
	podIdentifier string
	config        Config
}

// newOrchestrator creates a new ShardOrchestrator with the given configuration.
func newOrchestrator(cfg Config) *ShardOrchestrator {
	ringCfg := consistent.Config{
		PartitionCount:    cfg.PartitionCount,
		ReplicationFactor: cfg.ReplicationFactor,
		Load:              cfg.Load,
		Hasher:            blake3Hasher{},
	}

	ring := consistent.New(nil, ringCfg)

	// Add self to the ring immediately
	ring.Add(member(cfg.PodName))
	MembersAdded.Inc()

	return &ShardOrchestrator{
		ring:          ring,
		podIdentifier: cfg.PodName,
		config:        cfg,
	}
}

// Owns returns true if this pod should handle the resource with the given key.
// The key is typically the resource name or namespace/name.
func (s *ShardOrchestrator) Owns(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	owner := s.ring.LocateKey([]byte(key))
	return owner.String() == s.podIdentifier
}

// LocateOwner returns the pod identifier that owns the given key.
func (s *ShardOrchestrator) LocateOwner(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	owner := s.ring.LocateKey([]byte(key))
	return owner.String()
}

// MemberCount returns the current number of members in the hash ring.
func (s *ShardOrchestrator) MemberCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.ring.GetMembers())
}

// PodIdentifier returns this pod's identifier.
func (s *ShardOrchestrator) PodIdentifier() string {
	return s.podIdentifier
}

// GetMembers returns the current members in the ring.
func (s *ShardOrchestrator) GetMembers() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	members := s.ring.GetMembers()
	result := make([]string, len(members))
	for i, m := range members {
		result[i] = m.String()
	}
	return result
}

// syncMembers updates the hash ring with the current active members.
// This is called by the LeaseWatcher when membership changes.
func (s *ShardOrchestrator) syncMembers(members []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	logger := log.Log.WithName("sharding")

	// Build sets for comparison
	current := make(map[string]struct{})
	for _, m := range s.ring.GetMembers() {
		current[m.String()] = struct{}{}
	}

	incoming := make(map[string]struct{})
	for _, m := range members {
		incoming[m] = struct{}{}
	}

	// Find members to add
	var added []string
	for m := range incoming {
		if _, exists := current[m]; !exists {
			added = append(added, m)
		}
	}

	// Find members to remove
	var removed []string
	for m := range current {
		if _, exists := incoming[m]; !exists {
			removed = append(removed, m)
		}
	}

	// No changes
	if len(added) == 0 && len(removed) == 0 {
		return
	}

	// Capture partition ownership before changes
	partitionOwnersBefore := s.getPartitionOwners()

	// Apply changes
	for _, m := range added {
		logger.Info("Adding member to ring", "member", m)
		s.ring.Add(member(m))
		MembersAdded.Inc()
		RingChangeEvents.WithLabelValues("member_added").Inc()
	}

	for _, m := range removed {
		logger.Info("Removing member from ring", "member", m)
		s.ring.Remove(m)
		MembersRemoved.Inc()
		RingChangeEvents.WithLabelValues("member_removed").Inc()
	}

	// Capture partition ownership after changes
	partitionOwnersAfter := s.getPartitionOwners()

	// Count relocated partitions
	relocated := 0
	for partID, oldOwner := range partitionOwnersBefore {
		if newOwner, exists := partitionOwnersAfter[partID]; exists && oldOwner != newOwner {
			relocated++
		}
	}

	// Update metrics
	PartitionsRelocated.Add(float64(relocated))
	LastPartitionsRelocated.Set(float64(relocated))
	if s.config.PartitionCount > 0 {
		percent := float64(relocated) / float64(s.config.PartitionCount) * 100
		LastPartitionsRelocatedPercent.Set(percent)
		logger.Info("Ring membership changed",
			"added", len(added),
			"removed", len(removed),
			"partitionsRelocated", relocated,
			"partitionsRelocatedPercent", percent,
		)
	}
}

// getPartitionOwners returns a map of partitionID -> owner for all partitions.
func (s *ShardOrchestrator) getPartitionOwners() map[int]string {
	owners := make(map[int]string, s.config.PartitionCount)
	for i := 0; i < s.config.PartitionCount; i++ {
		owner := s.ring.GetPartitionOwner(i)
		if owner != nil {
			owners[i] = owner.String()
		}
	}
	return owners
}
