#!/bin/bash
#
# Builds the operator's controller and sidecar images. The manager's env var
# with the sidecar app image is also replaced with the new name, which is set
# using the value from env var SIDECAR_IMG. It will be defaulted to DEFAULT_SIDECAR_IMG
# if that env var is not found.
#
# This script is intended to be used in CI workflows to create loca/test images just for
# testing purposes, but can also be used to generate images for new official releases. In
# that case, the following env vars should have been exported before running this script:
#   VERSION : The desired new/test version tag that will be applied to both images.
#   IMG     : The image for the controller's container.
#

# Bash settings: display (expanded) commands and fast exit on first error.
set -o xtrace
set -o errexit

DEFAULT_TEST_VERSION="0.0.1-test"
DEFAULT_SIDECAR_IMG="ci-certsuite-op-sidecar:v$DEFAULT_TEST_VERSION"
DEFAULT_IMG="ci-certsuite-op:v$DEFAULT_TEST_VERSION"

export VERSION="${VERSION:-$DEFAULT_TEST_VERSION}"
export SIDECAR_IMG="${SIDECAR_IMG:-$DEFAULT_SIDECAR_IMG}"
export IMG="${IMG:-$DEFAULT_IMG}"

# step: Build manifests and controller app
make build

# step: Run tests
make test

# step: Build sidecar app
docker build --no-cache -t "${SIDECAR_IMG}" -f certsuite-sidecar/Dockerfile .

# Local install kustomize app that is needed to edit/patch the kustomization.yaml
make kustomize

# step: Build docker image for the controller. This will use IMG pointing to local docker image for the controller, which
#       will be pushed to kind with "docker load docker-image $IMG"
make docker-build
