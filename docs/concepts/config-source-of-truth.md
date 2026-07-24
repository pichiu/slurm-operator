# Configuration Source of Truth

## Table of Contents

<!-- mdformat-toc start --slug=github --no-anchors --maxlevel=6 --minlevel=1 -->

- [Configuration Source of Truth](#configuration-source-of-truth)
  - [Table of Contents](#table-of-contents)
  - [Overview](#overview)
  - [What the Operator Treats as Source of Truth](#what-the-operator-treats-as-source-of-truth)
  - [Two Paths for Partition Configuration](#two-paths-for-partition-configuration)
  - [Out-of-Band Resources and Upgrades](#out-of-band-resources-and-upgrades)
  - [Recommendations](#recommendations)

<!-- mdformat-toc end -->

## Overview

A common question when managing a Slurm cluster with this operator is: if a
`NodeSet` or partition is added *out-of-band* (created directly via `kubectl` or
an external platform, rather than through the Helm chart's `values.yaml`), does
it take effect, and what happens to it on the next `helm upgrade`?

The short answer, grounded in the operator's code, is:

- The operator's source of truth is the set of **Custom Resources (CRs) present
  in the cluster**, not `values.yaml`. Helm is simply one of the possible writers
  of those CRs.
- An out-of-band `NodeSet` (and its embedded partition) **is picked up
  automatically** and rendered into `slurm.conf`.
- The real thing an operator has to manage themselves is **drift between
  externally-created resources and what the Helm chart expects** — chiefly name
  collisions, and partitions declared through the chart's `partitions:` map.

This document explains the mechanism so that the trade-offs are explicit.

## What the Operator Treats as Source of Truth

When the Controller reconciles, it builds `slurm.conf` from a **live list of all
`NodeSet` CRs in the namespace**, filtered only by which Controller they
reference — there is no check for who created the resource, no Helm
`app.kubernetes.io/managed-by` label check, and no owner check:

```go
// internal/utils/refresolver/refresolver.go
list := &slinkyv1beta1.NodeSetList{}
r.reader.List(ctx, list, client.InNamespace(controller.Namespace)) // lists ALL NodeSet CRs
for _, item := range list.Items {
    if IsKeyMatch(item.Spec.ControllerRef.Name, controller) {       // filtered only by controllerRef
        out.Items = append(out.Items, item)
    }
}
```

The rendered NodeSet and partition lines come from
`buildNodeSetConf()` in `internal/builder/controllerbuilder/controller_config.go`,
which iterates every NodeSet returned above and emits a `NodeSet=` line plus,
when `partition.enabled` is `true`, a `PartitionName=` line.

The loop is closed by the Controller watching NodeSets
(`internal/controller/controller/controller_controller.go`, `Watches(&NodeSet{})`):
any NodeSet create/update/delete re-triggers a Controller reconcile, which
regenerates the `slurm.conf` ConfigMap (the Controller `Owns` that ConfigMap).

**Consequence:** applying a `NodeSet` CR out-of-band — one whose `controllerRef`
points at the Controller, with `partition.enabled: true` — causes the operator
to include it in `slurm.conf` automatically, with no involvement from Helm.

## Two Paths for Partition Configuration

There are two distinct ways a partition can reach `slurm.conf`, and they behave
oppositely with respect to out-of-band management:

| Path | Source | How it reaches `slurm.conf` | On `helm upgrade` |
| --- | --- | --- | --- |
| **B: NodeSet-embedded partition** | `NodeSetSpec.Partition` (`api/v1beta1/nodeset_types.go`) | Rendered live by the operator from the NodeSet list (`buildNodeSetConf`) | The CR object is **not deleted** (the chart sets no ownerReference and does no pruning), so it keeps being rendered |
| **A: Chart `partitions:` map** | Rendered by the Helm helper `slurm.controller.extraConf` (`helm/slurm/templates/controller/_helpers.tpl`) into `Controller.spec.extraConf` | The operator emits `controller.Spec.ExtraConf` verbatim (`controller_config.go`) | `helm upgrade` recomputes `extraConf` from `values.partitions` and **overwrites** the Controller spec |

Path A produces static text baked into the Controller CR at Helm template time;
Path B is reconciled live from separate CR objects.

## Out-of-Band Resources and Upgrades

Combining the above:

- **Out-of-band `NodeSet` CRs (Path B) survive upgrades.** The chart does not
  own them and does not prune them, and the operator keeps rendering them. They
  are not silently dropped on `helm upgrade`. The main hazard is a **name
  collision** with a NodeSet the chart later generates.
- **A partition added by hand-editing `Controller.spec.extraConf` (Path A) does
  not survive.** The next `helm upgrade` recomputes `extraConf` from
  `values.partitions` and replaces the Controller spec, so the manual line
  disappears.
- There is operator-level config merging where it matters:
  `BuildMergedConfig()` (`internal/builder/common/common.go`) merges
  operator-required parameters with the user's `extraConf`. What the operator
  does *not* do is merge Helm-managed values with out-of-band values inside
  `values.yaml`.
- The operator has **no runtime dependency on Helm** (Helm only appears under
  `test/e2e`). The CRs are the real API, so managing the cluster by applying CRs
  directly — without the Helm chart — is fully supported.

## Recommendations

- If you manage `NodeSet`/partition resources with an external platform, keep
  their identity (names) from colliding with resources the Slurm chart generates
  from `values.yaml`.
- Prefer expressing a partition as a **NodeSet-embedded partition (Path B)** when
  it is managed out-of-band, since it is reconciled from a durable CR object
  rather than from `Controller.spec.extraConf`.
- Treat `values.yaml` as the source of truth for anything the chart manages: if
  you want an out-of-band addition to persist across a chart-driven change,
  record it in `values.yaml` as well, or manage the whole cluster via CRs
  without the chart.
