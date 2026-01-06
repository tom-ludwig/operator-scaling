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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// KeyFunc extracts the sharding key from an object.
// By default, the object's name is used.
type KeyFunc func(obj client.Object) string

// DefaultKeyFunc returns the object's name as the sharding key.
func DefaultKeyFunc(obj client.Object) string {
	return obj.GetName()
}

// NamespacedKeyFunc returns "namespace/name" as the sharding key.
// Use this when you have resources with the same name in different namespaces.
func NamespacedKeyFunc(obj client.Object) string {
	return obj.GetNamespace() + "/" + obj.GetName()
}

// NewShardingPredicate creates a predicate that filters events based on ownership.
// Only events for resources owned by this pod will pass through.
//
// Example usage:
//
//	ctrl.NewControllerManagedBy(mgr).
//	    For(&v1.MyResource{}).
//	    WithEventFilter(sharding.NewShardingPredicate(orchestrator, nil)).
//	    Complete(r)
func NewShardingPredicate(orchestrator *ShardOrchestrator, keyFunc KeyFunc) predicate.Predicate {
	if keyFunc == nil {
		keyFunc = DefaultKeyFunc
	}

	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return orchestrator.Owns(keyFunc(e.Object))
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return orchestrator.Owns(keyFunc(e.ObjectNew))
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return orchestrator.Owns(keyFunc(e.Object))
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return orchestrator.Owns(keyFunc(e.Object))
		},
	}
}
