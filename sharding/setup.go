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
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
)

// Sharding holds the components needed for operator sharding.
// Use Setup() to create and initialize all components.
type Sharding struct {
	// Orchestrator determines resource ownership via consistent hashing.
	// Use this in your controllers to check ownership.
	Orchestrator *ShardOrchestrator

	// Config is the configuration used for this sharding instance.
	Config Config
}

// Setup initializes the sharding system and registers all components with the manager.
// It returns a Sharding instance that provides access to the orchestrator for use in controllers.
//
// Example usage:
//
//	// In main.go, after creating the manager:
//	sharding, err := sharding.Setup(mgr, sharding.DefaultConfig())
//	if err != nil {
//	    setupLog.Error(err, "unable to setup sharding")
//	    os.Exit(1)
//	}
//
//	// Pass sharding.Orchestrator to your controllers:
//	if err := (&controller.MyReconciler{
//	    Client:       mgr.GetClient(),
//	    Scheme:       mgr.GetScheme(),
//	    Orchestrator: sharding.Orchestrator,
//	}).SetupWithManager(mgr); err != nil {
//	    setupLog.Error(err, "unable to create controller")
//	    os.Exit(1)
//	}
func Setup(mgr ctrl.Manager, cfg Config) (*Sharding, error) {
	// Create the orchestrator (consistent hash ring)
	orchestrator := newOrchestrator(cfg)

	// Create and register the lease manager (heartbeat)
	// Pass apiReader for pod lookups to avoid cache-related RBAC issues
	leaseManager := newLeaseManager(mgr.GetClient(), mgr.GetAPIReader(), cfg)
	if err := mgr.Add(leaseManager); err != nil {
		return nil, fmt.Errorf("unable to register lease manager: %w", err)
	}

	// Create and register the lease watcher (peer discovery)
	leaseWatcher := newLeaseWatcher(mgr.GetClient(), mgr.GetScheme(), orchestrator, cfg)
	if err := leaseWatcher.SetupWithManager(mgr); err != nil {
		return nil, fmt.Errorf("unable to setup lease watcher: %w", err)
	}

	return &Sharding{
		Orchestrator: orchestrator,
		Config:       cfg,
	}, nil
}

// SetupWithDefaults initializes sharding with default configuration.
// This is a convenience function equivalent to Setup(mgr, DefaultConfig()).
func SetupWithDefaults(mgr ctrl.Manager) (*Sharding, error) {
	return Setup(mgr, DefaultConfig())
}
