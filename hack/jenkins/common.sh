#!/bin/bash

# Copyright 2016 The Kubernetes Authors All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


# This script downloads the test files from the build bucket and makes some executable.

# The script expects the following env variables:
# OS: The operating system
# ARCH: The architecture
# DRIVER: the driver to use for the test
# CONTAINER_RUNTIME: the container runtime to use for the test
# EXTRA_START_ARGS: additional flags to pass into minikube start
# EXTRA_TEST_ARGS: additional flags to pass into go test
# JOB_NAME: the name of the logfile and check name to update on github

set -x

export MINIKUBE_SUPPRESS_DOCKER_PERFORMANCE=true

readonly TIMEOUT=${1:-300m}
readonly OS_ARCH="${OS}-${ARCH}"
readonly TEST_OUT="testout.txt"
readonly JSON_OUT="test.json"

echo ">> Starting at $(date)"
echo ""
echo "driver:    ${DRIVER}"
echo "runtime:   ${CONTAINER_RUNTIME}"
# Setting KUBECONFIG prevents the version check from erroring out due to permission issues
echo "kubectl:   $(env KUBECONFIG=${TEST_HOME} kubectl version --client --short=true)"

cp -r test/integration/testdata/ testdata/

# Set the executable bit on the e2e binary and out binary
export MINIKUBE_BIN="out/minikube-${OS_ARCH}"
export E2E_BIN="out/e2e-${OS_ARCH}"

e2e_start_time="$(date -u +%s)"
echo ""
echo ">> Starting ${E2E_BIN} at $(date)"
set -x

if test -f "${TEST_OUT}"; then
  rm "${TEST_OUT}" || true # clean up previous runs of same build
fi
touch "${TEST_OUT}"

if [ ! -z "${CONTAINER_RUNTIME}" ]
then
    EXTRA_START_ARGS="${EXTRA_START_ARGS} --container-runtime=${CONTAINER_RUNTIME}"
fi

if test -f "${JSON_OUT}"; then
  rm "${JSON_OUT}" || true # clean up previous runs of same build
fi

touch "${JSON_OUT}"

gotestsum --jsonfile "${JSON_OUT}" -f standard-verbose --raw-command -- \
  go tool test2json -t \
  ${E2E_BIN} \
    -minikube-start-args="--driver=${DRIVER} ${EXTRA_START_ARGS}" \
    -test.timeout=${TIMEOUT} -test.v \
    ${EXTRA_TEST_ARGS} \
    -binary="${MINIKUBE_BIN}" 2>&1 \
  | tee "${TEST_OUT}"

