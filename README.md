# Operator Scaling

Horizontal scaling for Kubernetes operators using consistent hashing.

Instead of leader election (one pod does all work), this package distributes resources across all replicas. Each resource is deterministically assigned to exactly one pod.

## How It Works

The core of this package is **Consistent Hashing with Bounded Loads**, based on the Google research paper:
[Consistent Hashing with Bounded Loads](https://research.google/pubs/consistent-hashing-with-bounded-loads/) (Mirrokni et al., 2018)

This algorithm ensures:
- Minimal redistribution when nodes join/leave
- Even load distribution across replicas (bounded by a configurable load factor)
- Deterministic assignment of resources to pods

Implementation powered by [github.com/buraksezer/consistent](https://github.com/buraksezer/consistent).

## Installation

```bash
go get github.com/tom-ludwig/operator-scaling
```

## Quick Start

```go
import "github.com/tom-ludwig/operator-scaling/sharding"

// In main.go
shard, err := sharding.SetupWithDefaults(mgr)

// In your controller
func (r *MyReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&v1.MyResource{}).
        WithEventFilter(sharding.NewShardingPredicate(r.Orchestrator, nil)).
        Complete(r)
}
```

## Configuration

```go
cfg := sharding.DefaultConfig().
    WithNamespace("my-operator").
    WithLeasePrefix("myop-").
    WithHashRing(1009, 20, 1.25)

shard, err := sharding.Setup(mgr, cfg)
```

## Requirements

**Environment variables** (in your Deployment):
```yaml
env:
- name: POD_NAME
  valueFrom:
    fieldRef:
      fieldPath: metadata.name
- name: POD_NAMESPACE
  valueFrom:
    fieldRef:
      fieldPath: metadata.namespace
```

**RBAC** for Lease objects:

Option 1: Add this marker anywhere in your code (kubebuilder will auto-generate RBAC):
```go
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete
```

Option 2: Add manually to `config/rbac/role.yaml`:
```yaml
- apiGroups: ["coordination.k8s.io"]
  resources: ["leases"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
```

## Important: Changing Hash Ring Parameters

Hash ring parameters (`PartitionCount`, `ReplicationFactor`, `Load`) **cannot be changed during a rolling update**.
 Different replicas would have different hash ring configurations, causing inconsistent resource ownership.

To change these parameters:
1. Scale down to 0 replicas
2. Update the configuration
3. Scale back up

This ensures all replicas start with the same hash ring configuration.

## License

Apache 2.0