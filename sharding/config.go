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
	"os"
	"time"
)

// Config holds the configuration for the sharding system.
type Config struct {
	// PodName is the unique identifier for this pod.
	// Defaults to POD_NAME environment variable.
	PodName string

	// Namespace is the namespace where leases will be created.
	// Defaults to POD_NAMESPACE environment variable, or "default" if not set.
	Namespace string

	// LeasePrefix is the prefix for lease names.
	// Full lease name will be "{LeasePrefix}{PodName}".
	// Defaults to "shard-".
	LeasePrefix string

	// LeaseLabelKey is the label key used to identify sharding leases.
	// Defaults to "operator.sharding".
	LeaseLabelKey string

	// LeaseLabelValue is the label value used to identify sharding leases.
	// Defaults to "member".
	LeaseLabelValue string

	// HeartbeatInterval is how often the lease is renewed.
	// Defaults to 10 seconds.
	HeartbeatInterval time.Duration

	// LeaseDuration is how long a lease is valid without renewal.
	// Should be at least 3x HeartbeatInterval.
	// Defaults to 30 seconds.
	LeaseDuration time.Duration

	// PartitionCount is the number of partitions in the hash ring.
	// Higher values allow more replicas but use more memory.
	// Should be a prime number for better distribution.
	// Defaults to 509 (supports up to ~50 replicas).
	PartitionCount int

	// ReplicationFactor determines how many virtual nodes each member has.
	// Higher values provide better load distribution when members fail.
	// Defaults to 20.
	ReplicationFactor int

	// Load is the maximum load factor for each member.
	// Members with load above this threshold won't receive new keys.
	// Defaults to 1.25.
	Load float64
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	podName := os.Getenv("POD_NAME")
	if podName == "" {
		// Generate a unique name for local development
		podName = "dev-" + randomString(8)
	}

	namespace := os.Getenv("POD_NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}

	return Config{
		PodName:           podName,
		Namespace:         namespace,
		LeasePrefix:       "shard-",
		LeaseLabelKey:     "operator.sharding",
		LeaseLabelValue:   "member",
		HeartbeatInterval: 10 * time.Second,
		LeaseDuration:     30 * time.Second,
		PartitionCount:    509,
		ReplicationFactor: 20,
		Load:              1.25,
	}
}

// WithPodName sets the pod name.
func (c Config) WithPodName(name string) Config {
	c.PodName = name
	return c
}

// WithNamespace sets the namespace.
func (c Config) WithNamespace(ns string) Config {
	c.Namespace = ns
	return c
}

// WithLeasePrefix sets the lease prefix.
func (c Config) WithLeasePrefix(prefix string) Config {
	c.LeasePrefix = prefix
	return c
}

// WithLeaseLabels sets the lease label key and value.
func (c Config) WithLeaseLabels(key, value string) Config {
	c.LeaseLabelKey = key
	c.LeaseLabelValue = value
	return c
}

// WithHeartbeat sets the heartbeat interval and lease duration.
// leaseDuration should be at least 3x heartbeatInterval.
func (c Config) WithHeartbeat(interval, duration time.Duration) Config {
	c.HeartbeatInterval = interval
	c.LeaseDuration = duration
	return c
}

// WithHashRing sets the hash ring configuration.
func (c Config) WithHashRing(partitionCount, replicationFactor int, load float64) Config {
	c.PartitionCount = partitionCount
	c.ReplicationFactor = replicationFactor
	c.Load = load
	return c
}

// randomString generates a random string for dev mode pod names.
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
		time.Sleep(time.Nanosecond) // Ensure different values
	}
	return string(b)
}
