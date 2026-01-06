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

/*
Package sharding provides horizontal scaling for Kubernetes operators using consistent hashing.

Instead of using leader election (where only one pod does all the work), this package
distributes resources across all operator replicas using consistent hashing. Each resource
is deterministically assigned to exactly one replica, enabling true horizontal scaling.

# How It Works

The core is Consistent Hashing with Bounded Loads, based on the Google research paper:
https://research.google/pubs/consistent-hashing-with-bounded-loads/

Implementation powered by github.com/buraksezer/consistent.

  - Each operator pod maintains a Lease object as a heartbeat
  - Pods discover each other by watching Lease objects
  - A consistent hash ring determines which pod owns each resource
  - When pods are added/removed, minimal resources are redistributed

# Quick Start

In your main.go, after creating the manager:

	import "github.com/tom-ludwig/operator-scaling/sharding"

	func main() {
	    // Create manager with leader election DISABLED
	    mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
	        LeaderElection: false, // Required - all replicas must be active
	        // ...
	    })

	    // Setup sharding
	    shard, err := sharding.SetupWithDefaults(mgr)
	    if err != nil {
	        setupLog.Error(err, "unable to setup sharding")
	        os.Exit(1)
	    }

	    // Pass orchestrator to your controllers
	    if err := (&controller.MyReconciler{
	        Client:       mgr.GetClient(),
	        Scheme:       mgr.GetScheme(),
	        Orchestrator: shard.Orchestrator,
	    }).SetupWithManager(mgr); err != nil {
	        setupLog.Error(err, "unable to create controller")
	        os.Exit(1)
	    }

	    // ... start manager ...
	}

In your controller:

	type MyReconciler struct {
	    client.Client
	    Scheme       *runtime.Scheme
	    Orchestrator *sharding.ShardOrchestrator
	}

	func (r *MyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	    return ctrl.NewControllerManagedBy(mgr).
	        For(&v1.MyResource{}).
	        WithEventFilter(sharding.NewShardingPredicate(r.Orchestrator, nil)).
	        Complete(r)
	}

# Configuration

Use DefaultConfig() and chain With* methods:

	cfg := sharding.DefaultConfig().
	    WithNamespace("my-operator").
	    WithLeasePrefix("myop-").
	    WithHashRing(1009, 20, 1.25)

	shard, err := sharding.Setup(mgr, cfg)

# Requirements

Disable leader election (all replicas must be active):

	LeaderElection: false

Environment variables in your Deployment:

	env:
	- name: POD_NAME
	  valueFrom:
	    fieldRef:
	      fieldPath: metadata.name
	- name: POD_NAMESPACE
	  valueFrom:
	    fieldRef:
	      fieldPath: metadata.namespace

RBAC (add these markers to auto-generate):

	// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete
	// +kubebuilder:rbac:groups="",resources=pods,verbs=get

The pods/get permission sets owner references for automatic lease cleanup.
If unavailable, leases still expire naturally (~30s).

# Changing Hash Ring Parameters

Hash ring parameters (PartitionCount, ReplicationFactor, Load) cannot be changed
during a rolling update. Different replicas would have different configurations.

To change these parameters: scale to 0, update config, scale back up.

# Metrics

  - sharding_ring_member_count: Current members in the ring
  - sharding_ring_members_added_total: Members added over time
  - sharding_ring_members_removed_total: Members removed over time
  - sharding_partitions_relocated_total: Partitions that changed ownership
  - sharding_lease_heartbeats_total: Successful heartbeats
  - sharding_lease_heartbeat_failures_total: Failed heartbeats (alert on this)
*/
package sharding
