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

	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// LeaseManager manages the pod's lease heartbeat.
// It creates and periodically renews a Lease object to signal liveness to other pods.
type LeaseManager struct {
	client    client.Client
	apiReader client.Reader // Direct API access (bypasses cache, avoids cluster-wide RBAC)
	config    Config
	ownerRef  *metav1.OwnerReference
}

// newLeaseManager creates a new LeaseManager.
// apiReader is used for pod lookups to avoid cache-related RBAC issues.
func newLeaseManager(c client.Client, apiReader client.Reader, cfg Config) *LeaseManager {
	return &LeaseManager{
		client:    c,
		apiReader: apiReader,
		config:    cfg,
	}
}

// Start runs the heartbeat loop. Implements manager.Runnable.
func (l *LeaseManager) Start(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("lease-manager")

	// Try to fetch pod for owner reference (for garbage collection)
	if err := l.ensureOwnerRef(ctx); err != nil {
		logger.Info("Unable to fetch pod for ownerRef (this is normal in local dev)", "error", err)
	}

	// Run heartbeat loop
	wait.UntilWithContext(ctx, func(ctx context.Context) {
		l.heartbeat(ctx)
	}, l.config.HeartbeatInterval)

	return nil
}

// ensureOwnerRef retrieves the current Pod to set it as the Lease owner.
// This ensures the lease is garbage collected when the pod is deleted.
// Uses apiReader (direct API call) to avoid cache-related RBAC issues -
// only requires "get" permission on pods in the operator's namespace.
func (l *LeaseManager) ensureOwnerRef(ctx context.Context) error {
	if l.ownerRef != nil || l.config.PodName == "" {
		return nil
	}

	var pod corev1.Pod
	if err := l.apiReader.Get(ctx, types.NamespacedName{
		Namespace: l.config.Namespace,
		Name:      l.config.PodName,
	}, &pod); err != nil {
		return err
	}

	l.ownerRef = metav1.NewControllerRef(&pod, corev1.SchemeGroupVersion.WithKind("Pod"))
	return nil
}

// heartbeat creates or updates the pod's lease.
func (l *LeaseManager) heartbeat(ctx context.Context) {
	logger := log.FromContext(ctx).WithName("lease-manager")

	lease := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      l.config.LeasePrefix + l.config.PodName,
			Namespace: l.config.Namespace,
			Labels: map[string]string{
				l.config.LeaseLabelKey: l.config.LeaseLabelValue,
			},
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, l.client, lease, func() error {
		now := metav1.NowMicro()
		lease.Spec.HolderIdentity = &l.config.PodName
		lease.Spec.RenewTime = &now

		durationSeconds := int32(l.config.LeaseDuration.Seconds())
		lease.Spec.LeaseDurationSeconds = &durationSeconds

		// Ensure labels are set (in case lease was created without them)
		if lease.Labels == nil {
			lease.Labels = make(map[string]string)
		}
		lease.Labels[l.config.LeaseLabelKey] = l.config.LeaseLabelValue

		// Set owner reference for garbage collection
		if l.ownerRef != nil {
			lease.OwnerReferences = []metav1.OwnerReference{*l.ownerRef}
		}

		return nil
	})

	if err != nil {
		logger.Error(err, "Failed to update heartbeat lease")
		LeaseHeartbeatFailures.Inc()
	} else {
		logger.V(1).Info("Heartbeat sent", "operation", op)
		LeaseHeartbeats.Inc()
	}
}
