// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package defaults

import (
	"k8s.io/utils/ptr"

	slinkyv1beta1 "github.com/SlinkyProject/slurm-operator/api/v1beta1"
)

// Default values for LoginSet Spec fields when unspecified.
const (
	DefaultLoginSetReplicas int32 = 1
)

// SetLoginSetDefaults applies default values for LoginSet spec fields.
// The Kubernetes API does not always apply CRD schema defaults (e.g. unless the
// parent object is given). Calling this on a copy when we read the LoginSet
// ensures the controller always sees defaulted values. The stored object in the
// API server is unchanged.
//
// Call this on a copy of the LoginSet (e.g. after DeepCopy in the controller)
// before using the spec.
func SetLoginSetDefaults(loginset *slinkyv1beta1.LoginSet) {
	if loginset == nil {
		return
	}
	s := &loginset.Spec

	if s.Replicas == nil {
		s.Replicas = ptr.To(DefaultLoginSetReplicas)
	}
}
