#!/usr/bin/env bash
# SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

# Allow unprivileged users to mount SIF images via squashfuse/fuse2fs.
# Some hosts create /dev/fuse as 0600; this only affects the container's /dev.
if [ -c /dev/fuse ]; then
	chmod 666 /dev/fuse 2>/dev/null || true
fi
