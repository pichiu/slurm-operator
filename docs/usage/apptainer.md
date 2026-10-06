# Apptainer Guide

## Table of Contents

<!-- mdformat-toc start --slug=github --no-anchors --maxlevel=6 --minlevel=1 -->

- [Apptainer Guide](#apptainer-guide)
  - [Table of Contents](#table-of-contents)
  - [Overview](#overview)
  - [Pre-requisites](#pre-requisites)
  - [Build Images](#build-images)
  - [Configure](#configure)
    - [Images](#images)
    - [Slurm OCI Runtime](#slurm-oci-runtime)
  - [Test](#test)
    - [Apptainer in Job Steps](#apptainer-in-job-steps)
    - [Slurm Container Jobs](#slurm-container-jobs)
    - [GPUs](#gpus)
  - [Caveats](#caveats)

<!-- mdformat-toc end -->

## Overview

This guide tells how to configure your Slurm cluster to run containerized jobs
with [Apptainer] (formerly Singularity), as an alternative to [pyxis] and
[enroot].

Apptainer can be used in two ways:

1. Directly, by calling `apptainer exec` from within a job step.
1. Through Slurm's native [container support][slurm-containers] (`--container`),
   with Apptainer configured as the OCI runtime in [oci.conf].

Both rely on Apptainer running unprivileged with user namespaces. The setuid
installation (`apptainer-suid`) is neither installed nor needed.

## Pre-requisites

The Kubernetes nodes running slurmd pods must allow unprivileged user
namespaces. On hosts using AppArmor (e.g. Ubuntu 24.04+), this may require:

```bash
sudo sysctl -w kernel.apparmor_restrict_unprivileged_userns=0
sudo sysctl -w kernel.apparmor_restrict_unprivileged_unconfined=0
```

The slurmd container is already privileged, which gives it access to `/dev/fuse`
for mounting SIF images. The login container is not privileged by default.

## Build Images

There are no published Apptainer images. Build them from `Dockerfile.apptainer`
in the `hack/apptainer/` directory of this repository. The images extend the
stock `slurmd` and `login` images with:

- `apptainer` from the official [Apptainer PPA][apptainer-ppa].
- `squashfuse`, `fuse2fs` and `fuse-overlayfs` for unprivileged image mounts.
- `nvidia-container-toolkit` for `apptainer --nvccli`.
- `/usr/local/bin/apptainer-oci-run`, an [oci.conf] `RunTimeRun` wrapper (slurmd
  only).
- Entrypoint scripts that remount `/proc` and `/tmp` (as the pyxis images do)
  and make `/dev/fuse` usable by job users.

```bash
cd hack/apptainer
docker build -f Dockerfile.apptainer --target slurmd-apptainer \
  -t <registry>/slurmd-apptainer:26.05-ubuntu26.04 .
docker build -f Dockerfile.apptainer --target login-apptainer \
  -t <registry>/login-apptainer:26.05-ubuntu26.04 .
```

Use `--build-arg SLURM_TAG=<tag>` to build from a different base image tag. Only
Ubuntu base images are supported.

## Configure

### Images

Configure one or more NodeSets and the login pods to use the Apptainer images.

```yaml
loginsets:
  apptainer:
    login:
      image:
        repository: <registry>/login-apptainer
        tag: 26.05-ubuntu26.04
      securityContext:
        privileged: true
nodesets:
  apptainer:
    slurmd:
      image:
        repository: <registry>/slurmd-apptainer
        tag: 26.05-ubuntu26.04
    partition:
      enabled: true
```

The login container needs `securityContext.privileged=true` for users to run
`apptainer exec` or `apptainer build` there, otherwise Apptainer fails with
`Failed to create user namespace`. Leave it unprivileged if users only pull
images and submit jobs from it.

### Slurm OCI Runtime

To support `srun --container` and `sbatch --container`, configure `oci.conf`. It
is distributed to the slurmd and login pods via configless.

```yaml
configFiles:
  oci.conf: |
    IgnoreFileConfigJson=true
    CreateEnvFile=null
    EnvExclude="^(SLURM_CONF|SLURM_CONF_SERVER)="
    RunTimeEnvExclude="^(SLURM_CONF|SLURM_CONF_SERVER)="
    RunTimeRun="/usr/local/bin/apptainer-oci-run %r %e -- %@"
    RunTimeKill="kill -s SIGTERM %p"
    RunTimeDelete="kill -s SIGKILL %p"
```

With `IgnoreFileConfigJson=true`, the path given to `--container` is passed to
Apptainer as-is, so it can be either a SIF image or a rootfs directory; no OCI
bundle (`config.json`) is required.

The `apptainer-oci-run` wrapper runs `apptainer exec --userns` with the job
step's environment, and binds the slurmd spool directory so that batch scripts
are reachable inside the container.

> [!NOTE]
> The [Slurm containers guide][slurm-containers] suggests
> `RunTimeRun="singularity exec --userns %r %@"`. This works only for simple
> `srun` steps: the step environment (e.g. `SLURM_PROCID`) is not passed to the
> container and batch scripts fail to start. Apptainer's `--env-file` is not a
> substitute, as it evaluates the file as a shell script and fails on values
> containing spaces.

## Test

### Apptainer in Job Steps

Pull an image (from a login pod, or within a job) and run it.

```console
$ apptainer pull alpine.sif docker://alpine:latest
$ srun --partition=apptainer apptainer exec alpine.sif grep PRETTY /etc/os-release
PRETTY_NAME="Alpine Linux v3.24"
```

> [!TIP]
> Point `APPTAINER_CACHEDIR` and `APPTAINER_TMPDIR` at a node-local or shared
> scratch volume, to avoid filling the user's home directory.

### Slurm Container Jobs

With [oci.conf](#slurm-oci-runtime) configured, request the container through
Slurm.

```console
$ srun --partition=apptainer --ntasks=2 --container=$HOME/alpine.sif /bin/sh -c 'echo task $SLURM_PROCID of $SLURM_NTASKS'
task 0 of 2
task 1 of 2
$ srun --partition=apptainer --container=$HOME/alpine.sif /bin/grep PRETTY /etc/os-release
PRETTY_NAME="Alpine Linux v3.24"
$ sbatch --partition=apptainer --container=$HOME/alpine.sif --wrap 'grep PRETTY /etc/os-release'
Submitted batch job 3
```

Apptainer settings can be changed per job through `APPTAINER_*` environment
variables, which the wrapper passes through to `apptainer`.

```bash
export APPTAINER_BIND=/data
srun --partition=apptainer --container=$HOME/alpine.sif ls /data
```

### GPUs

Request GPUs from Slurm and enable Apptainer's NVIDIA support.

> [!NOTE]
> GPU support has not yet been validated with these images.

```bash
srun --partition=apptainer --gpus=1 apptainer exec --nv pytorch.sif nvidia-smi
APPTAINER_NV=1 srun --partition=apptainer --gpus=1 --container=$HOME/pytorch.sif nvidia-smi
```

## Caveats

- `srun` resolves the command against the `PATH` of the submitting host before
  the container starts, so a command found at `/usr/bin/grep` on the host is run
  as `/usr/bin/grep` in the container. Use paths valid inside the container
  (e.g. `/bin/sh`) when the image layout differs from the host.
- The pyxis and Apptainer images can coexist in one cluster, but use partitions
  and/or features to steer jobs to nodes with the runtime they need.
- Some hosts create `/dev/fuse` with mode `0600`. The image entrypoint changes
  it to `0666` inside the container only, so that job users can mount SIF
  images.

<!-- Links -->

[apptainer]: https://apptainer.org/
[apptainer-ppa]: https://launchpad.net/~apptainer/+archive/ubuntu/ppa
[enroot]: https://github.com/NVIDIA/enroot
[oci.conf]: https://slurm.schedmd.com/oci.conf.html
[pyxis]: https://github.com/NVIDIA/pyxis
[slurm-containers]: https://slurm.schedmd.com/containers.html
