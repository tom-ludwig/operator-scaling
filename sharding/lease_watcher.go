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
	"context"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// LeaseWatcher watches Lease objects and syncs the hash ring with active members.
type LeaseWatcher struct {
	client.Client
	Scheme       *runtime.Scheme
	orchestrator *ShardOrchestrator
	config       Config
}

func newLeaseWatcher(c client.Client, scheme *runtime.Scheme, orchestrator *ShardOrchestrator, cfg Config) *LeaseWatcher {
	return &LeaseWatcher{
		Client:       c,
		Scheme:       scheme,
		orchestrator: orchestrator,
		config:       cfg,
	}
}

// Reconcile handles lease changes and syncs the hash ring.
func (r *LeaseWatcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Fetch all leases that match our label selector
	var list coordinationv1.LeaseList

	selector := labels.SelectorFromSet(map[string]string{
		r.config.LeaseLabelKey: r.config.LeaseLabelValue,
	})

	if err := r.List(ctx, &list, &client.ListOptions{
		Namespace:     req.Namespace,
		LabelSelector: selector,
	}); err != nil {
		return ctrl.Result{}, err
	}

	// Filter to active (non-expired) members
	var activeMembers []string
	now := time.Now()

	for _, lease := range list.Items {
		if lease.Spec.RenewTime == nil {
			continue
		}

		// Check if lease is expired (allow a small buffer for clock skew)
		duration := time.Duration(*lease.Spec.LeaseDurationSeconds) * time.Second
		if now.Sub(lease.Spec.RenewTime.Time) < (duration + 5*time.Second) {
			if lease.Spec.HolderIdentity != nil {
				activeMembers = append(activeMembers, *lease.Spec.HolderIdentity)
			}
		}
	}

	// Update ring membership
	r.orchestrator.syncMembers(activeMembers)
	RingMemberCount.Set(float64(len(activeMembers)))

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LeaseWatcher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coordinationv1.Lease{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				return r.isShardLease(e.Object.GetLabels())
			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				return r.isShardLease(e.ObjectNew.GetLabels())
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				return r.isShardLease(e.Object.GetLabels())
			},
			GenericFunc: func(e event.GenericEvent) bool {
				return r.isShardLease(e.Object.GetLabels())
			},
		}).
		Complete(r)
}

// isShardLease checks if the lease belongs to our sharding system.
func (r *LeaseWatcher) isShardLease(objectLabels map[string]string) bool {
	return objectLabels[r.config.LeaseLabelKey] == r.config.LeaseLabelValue
}
