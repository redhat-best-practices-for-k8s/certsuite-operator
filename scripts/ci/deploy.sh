#!/bin/bash
#
# This script deploys a recently built operator in a kind cluster.
# Both the operator's controller and the sidecar app images are
# preloaded into the kind cluster's nodes to avoid the need of
# uploading the images to an external registry (quay/docker).
#
# The operator is deployed in the namespace set by env var CNF_CERTSUITE_OPERATOR_NAMESPACE
# or in the defaulted namespace "certsuite-operator" if that env var is not found.
#

# Bash settings: display (expanded) commands and fast exit on first error.
set -o xtrace
set -o errexit

DEFAULT_CNF_CERTSUITE_OPERATOR_NAMESPACE="certsuite-operator"
DEFAULT_TEST_VERSION="0.0.1-test"
DEFAULT_SIDECAR_IMG="ci-certsuite-op-sidecar:v$DEFAULT_TEST_VERSION"
DEFAULT_IMG="ci-certsuite-op:v$DEFAULT_TEST_VERSION"

CNF_CERTSUITE_OPERATOR_NAMESPACE=${CNF_CERTSUITE_OPERATOR_NAMESPACE:-$DEFAULT_CNF_CERTSUITE_OPERATOR_NAMESPACE}

export VERSION="${VERSION:-$DEFAULT_TEST_VERSION}"
export SIDECAR_IMG="${SIDECAR_IMG:-$DEFAULT_SIDECAR_IMG}"
export IMG="${IMG:-$DEFAULT_IMG}"

kind load docker-image "${SIDECAR_IMG}"
kind load docker-image "${IMG}"

# "make deploy" uses the IMG env var internally, and it needs to be exported.
# let's patch the installation namespace.
make kustomize

pushd config/default
  ../../bin/kustomize edit set namespace "${CNF_CERTSUITE_OPERATOR_NAMESPACE}"
popd

make deploy

# step: Wait for the controller's containers to be ready
oc wait --for=condition=ready pod --all=true -n certsuite-operator --timeout=2m