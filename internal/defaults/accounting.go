// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package defaults

import (
	slinkyv1beta1 "github.com/SlinkyProject/slurm-operator/api/v1beta1"
)

// Default values for Accounting Spec fields when unspecified.
const (
	DefaultAccountingStoragePort int    = 3306
	DefaultAccountingStorageDB   string = "slurm_acct_db"
)

// SetAccountingDefaults applies default values for Accounting spec fields.
// The Kubernetes API does not always apply CRD schema defaults (e.g. unless the
// parent object is given). Calling this on a copy when we read the Accounting
// ensures the controller always sees defaulted values. The stored object in the
// API server is unchanged.
//
// Call this on a copy of the Accounting (e.g. after DeepCopy in the controller)
// before using the spec.
func SetAccountingDefaults(accounting *slinkyv1beta1.Accounting) {
	if accounting == nil {
		return
	}
	s := &accounting.Spec
	// External is bool; zero value is false, which is the default.
	if s.StorageConfig.Port == 0 {
		s.StorageConfig.Port = DefaultAccountingStoragePort
	}
	if s.StorageConfig.Database == "" {
		s.StorageConfig.Database = DefaultAccountingStorageDB
	}
}
