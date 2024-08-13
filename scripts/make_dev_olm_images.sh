#!/bin/bash
#
# Developer's script to build all the operator's container images and push them to a custom quay account.
# Only VERSION (e.g. 0.0.4) and REPO (e.g. quay.io/myaccount/myop) env vars are needed.
# These images will be built and pushed to the appropriate registry:
# - sidecar app
# - controller
# - bundle
# - catalog
#
# These env vars should be exported to be used by this script:
# - VERSION: "without the v". E.g: "0.2.1", "1.0.1"
# - REPO: the docker registry repo for the operator, e.g: "quay.io/myaccount/my-operator"
#
# Only one image registry is needed as all the images will share the same name. The corresponding suffix will be
# appended to the version to distinguish them. For example, running this command:
#  VERSION=0.0.2 REPO=quay.io/myaccount/myop ./make_dev_olm_images.sh
# will make and upload these images:
#   quay.io/myaccount/myop:v0.0.1-controller
#   quay.io/myaccount/myop:v0.0.1-sidecar
#   quay.io/myaccount/myop:v0.0.1-bundle
#   quay.io/myaccount/myop:v0.0.1-catalog
#
# At the end of the script, the command to install this developer operator version will be shown, like:
#   OLM_INSTALL_IMG_CATALOG=quay.io/myaccount/myop:v0.0.1 make olm-install

# Fast exit on any returned error.
set -o errexit

# Check both VERSION and REPO env vars exist and are not empty.
if [ "${REPO}" == "" ]; then
  echo "ERROR: REPO env var is not exported or is empty."
  echo "E.g. REPO=quay.io/myaccount/myrepo"
  exit 1
fi

if [ "${VERSION}" == "" ]; then
  echo "ERROR: VERSION env var is not exported or is empty."
  echo "E.g. VERSION=0.0.1"
  exit 1
fi

BASE_URL=${REPO}:v${VERSION}
echo "Base docker repo for images: ${BASE_URL}"

export IMG=${BASE_URL}-controller
export SIDECAR_IMG=${BASE_URL}-sidecar
export BUNDLE_IMG=${BASE_URL}-bundle
export CATALOG_IMG=${BASE_URL}-catalog

echo "Building sidecar app's image..."
docker build --no-cache -t "${SIDECAR_IMG}" -f certsuite-sidecar/Dockerfile .

echo "Pushing sidecar app's image..."
docker push "${SIDECAR_IMG}"

echo "Building and pushing controller's image..."
make docker-build docker-push

echo "Removing all previous bundle's manifests..."
rm bundle/manifests/* || true

echo "Creating bundle skeleton..."
make bundle

echo "Building and pushing bundle image..."
make bundle-build bundle-push

echo "Building and pushing catalog image..."
make catalog-build catalog-push

echo
echo "The following container images have been successfully built and pushed into quay.io"
echo "${IMG}"
echo "${SIDECAR_IMG}"
echo "${BUNDLE_IMG}"
echo "${CATALOG_IMG}"
echo
echo "Use this command to deploy the operator through an OLM subscription:"
echo "  OLM_INSTALL_IMG_CATALOG=${CATALOG_IMG} make olm-install"