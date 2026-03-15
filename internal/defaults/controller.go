// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package defaults

import (
	"k8s.io/utils/ptr"

	slinkyv1beta1 "github.com/SlinkyProject/slurm-operator/api/v1beta1"
)

// Default values for Controller Spec fields when unspecified.
const (
	DefaultControllerPersistenceEnabled bool = true
)

// SetControllerDefaults applies default values for Controller spec fields.
// The Kubernetes API does not always apply CRD schema defaults (e.g. unless the
// parent object is given). Calling this on a copy when we read the Controller
// ensures the controller always sees defaulted values. The stored object in the
// API server is unchanged.
//
// Call this on a copy of the Controller (e.g. after DeepCopy in the controller)
// before using the spec.
func SetControllerDefaults(controller *slinkyv1beta1.Controller) {
	if controller == nil {
		return
	}
	s := &controller.Spec

	if s.Persistence.Enabled == nil {
		s.Persistence.Enabled = ptr.To(DefaultControllerPersistenceEnabled)
	}
}
