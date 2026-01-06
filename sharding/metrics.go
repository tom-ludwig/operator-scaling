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
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// MembersAdded tracks the total number of members added to the hash ring.
	MembersAdded = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "sharding_ring_members_added_total",
			Help: "Total number of members added to the consistent hash ring",
		},
	)

	// MembersRemoved tracks the total number of members removed from the hash ring.
	MembersRemoved = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "sharding_ring_members_removed_total",
			Help: "Total number of members removed from the consistent hash ring",
		},
	)

	// RingMemberCount tracks the current number of members in the hash ring.
	RingMemberCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "sharding_ring_member_count",
			Help: "Current number of members in the consistent hash ring",
		},
	)

	// RingChangeEvents tracks ring membership change events by type.
	RingChangeEvents = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sharding_ring_change_events_total",
			Help: "Total number of ring membership change events",
		},
		[]string{"event_type"}, // "member_added", "member_removed"
	)

	// PartitionsRelocated tracks total partitions that changed ownership.
	PartitionsRelocated = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "sharding_partitions_relocated_total",
			Help: "Total number of partitions that changed ownership due to ring membership changes",
		},
	)

	// LastPartitionsRelocated shows partitions moved in the most recent ring change.
	LastPartitionsRelocated = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "sharding_last_partitions_relocated",
			Help: "Number of partitions relocated in the last ring membership change",
		},
	)

	// LastPartitionsRelocatedPercent shows the percentage of partitions moved.
	LastPartitionsRelocatedPercent = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "sharding_last_partitions_relocated_percent",
			Help: "Percentage of partitions relocated in the last ring membership change",
		},
	)

	// LeaseHeartbeats tracks successful lease heartbeats.
	LeaseHeartbeats = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "sharding_lease_heartbeats_total",
			Help: "Total number of successful lease heartbeats",
		},
	)

	// LeaseHeartbeatFailures tracks failed lease heartbeats.
	LeaseHeartbeatFailures = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "sharding_lease_heartbeat_failures_total",
			Help: "Total number of failed lease heartbeats",
		},
	)
)

func init() {
	metrics.Registry.MustRegister(
		MembersAdded,
		MembersRemoved,
		RingMemberCount,
		RingChangeEvents,
		PartitionsRelocated,
		LastPartitionsRelocated,
		LastPartitionsRelocatedPercent,
		LeaseHeartbeats,
		LeaseHeartbeatFailures,
	)
}
