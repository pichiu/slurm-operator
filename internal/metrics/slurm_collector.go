// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"context"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/log"

	slurmtypes "github.com/SlinkyProject/slurm-client/pkg/types"

	"github.com/SlinkyProject/slurm-operator/internal/clientmap"
)

const (
	metricsNamespace = "slurm"
)

// SlurmCollector implements prometheus.Collector. It queries the Slurm REST API
// via each registered client in ClientMap and exposes job, node, and partition
// counts as Prometheus gauges, labeled by Kubernetes namespace, controller
// name, and Slurm state.
type SlurmCollector struct {
	clientMap *clientmap.ClientMap

	jobsDesc       *prometheus.Desc
	nodesDesc      *prometheus.Desc
	partitionsDesc *prometheus.Desc
}

func NewSlurmCollector(cm *clientmap.ClientMap) *SlurmCollector {
	labels := []string{"k8s_namespace", "controller", "state"}
	return &SlurmCollector{
		clientMap: cm,
		jobsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(metricsNamespace, "jobs", "total"),
			"Number of Slurm jobs in a given state, per cluster controller.",
			labels, nil,
		),
		nodesDesc: prometheus.NewDesc(
			prometheus.BuildFQName(metricsNamespace, "nodes", "total"),
			"Number of Slurm nodes in a given state, per cluster controller.",
			labels, nil,
		),
		partitionsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(metricsNamespace, "partitions", "total"),
			"Number of Slurm partitions in a given state, per cluster controller.",
			labels, nil,
		),
	}
}

// Describe implements prometheus.Collector.
func (c *SlurmCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.jobsDesc
	ch <- c.nodesDesc
	ch <- c.partitionsDesc
}

// Collect implements prometheus.Collector. It is called by Prometheus on each
// scrape and queries the Slurm REST API for current job/node/partition counts.
func (c *SlurmCollector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.Background()
	logger := log.FromContext(ctx).WithName("slurm-collector")

	for _, key := range c.clientMap.List() {
		slurmClient := c.clientMap.Get(key)
		if slurmClient == nil {
			continue
		}

		ns := key.Namespace
		name := key.Name

		jobList := &slurmtypes.V0044JobInfoList{}
		if err := slurmClient.List(ctx, jobList); err != nil {
			logger.Error(err, "failed to list jobs", "controller", key)
		} else {
			stateCounts := make(map[string]float64)
			for _, job := range jobList.Items {
				for state := range job.GetStateAsSet() {
					stateCounts[strings.ToLower(string(state))]++
				}
			}
			for state, count := range stateCounts {
				ch <- prometheus.MustNewConstMetric(c.jobsDesc, prometheus.GaugeValue, count, ns, name, state)
			}
		}

		nodeList := &slurmtypes.V0044NodeList{}
		if err := slurmClient.List(ctx, nodeList); err != nil {
			logger.Error(err, "failed to list nodes", "controller", key)
		} else {
			stateCounts := make(map[string]float64)
			for _, node := range nodeList.Items {
				for state := range node.GetStateAsSet() {
					stateCounts[strings.ToLower(string(state))]++
				}
			}
			for state, count := range stateCounts {
				ch <- prometheus.MustNewConstMetric(c.nodesDesc, prometheus.GaugeValue, count, ns, name, state)
			}
		}

		partitionList := &slurmtypes.V0044PartitionInfoList{}
		if err := slurmClient.List(ctx, partitionList); err != nil {
			logger.Error(err, "failed to list partitions", "controller", key)
		} else {
			stateCounts := make(map[string]float64)
			for _, partition := range partitionList.Items {
				for state := range partition.GetStateAsSet() {
					stateCounts[strings.ToLower(string(state))]++
				}
			}
			for state, count := range stateCounts {
				ch <- prometheus.MustNewConstMetric(c.partitionsDesc, prometheus.GaugeValue, count, ns, name, state)
			}
		}
	}
}
