# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

The certsuite-operator is a Kubernetes/OpenShift Operator scaffolded with operator-sdk that runs the [Certification Suite](https://github.com/redhat-best-practices-for-k8s/certsuite) container. It provides a declarative way to execute CNF/Cloud Native Functions certification tests via Custom Resources.

### How It Works

1. The operator registers a CRD called `CertsuiteRun` in the cluster
2. Users create a `CertsuiteRun` CR along with a ConfigMap (containing test configuration) and optionally a Secret (for preflight credentials)
3. When a CR is deployed, the operator creates a pod with two containers:
   - **certsuite container**: Runs the CNF certification test suite
   - **sidecar container**: Monitors test completion and updates the CR status with results
4. Test results are stored in the CR's status subresource and can be queried via kubectl/oc

### Custom Resource Definitions

- **CertsuiteRun**: Triggers a certification test run with configurable parameters (labels filter, log level, timeout, etc.)
- **CertsuiteConsolePlugin**: Manages the OpenShift console plugin for viewing results (optional)

## Build Commands

### Core Build Operations
```bash
make build              # Build manager binary (generates manifests, code, runs fmt and vet)
make docker-build       # Build controller container image
make docker-push        # Push controller image to registry
make docker-buildx      # Build and push multi-platform image (linux/arm64,amd64,s390x,ppc64le)
```

### Sidecar Build Operations
```bash
make sidecar-build      # Build sidecar container image
make sidecar-push       # Push sidecar image to registry
```

### Bundle and Catalog (OLM)
```bash
make bundle             # Generate OLM bundle manifests and validate
make bundle-build       # Build bundle image
make bundle-push        # Push bundle image
make catalog-build      # Build OLM catalog image
make catalog-push       # Push catalog image
make release            # Build and push all images (controller, sidecar, bundle, catalog)
```

### Code Generation
```bash
make manifests          # Generate CRDs, RBAC, and webhook configurations
make generate           # Generate DeepCopy methods for API types
make fmt                # Run go fmt
make vet                # Run go vet
```

## Test Commands

```bash
make test               # Run unit tests with coverage (uses envtest)
make test-e2e           # Run end-to-end tests against a Kind cluster
```

## Deployment Commands

### Install via OLM (Recommended)
```bash
export OLM_CATALOG=quay.io/redhat-best-practices-for-k8s/certsuite-operator-catalog:latest
export OLM_INSTALL_NAMESPACE=certsuite-operator
make olm-install        # Install operator via OLM subscription
make olm-uninstall      # Uninstall OLM-deployed operator
```

### Manual Deployment
```bash
# First install cert-manager:
kubectl apply -f https://github.com/jetstack/cert-manager/releases/latest/download/cert-manager.yaml

# Then deploy:
export IMG=<your-controller-image>
export SIDECAR_IMG=<your-sidecar-image>
make deploy             # Deploy controller to cluster
make undeploy           # Remove controller from cluster
```

### CRD Management
```bash
make install            # Install CRDs into cluster
make uninstall          # Remove CRDs from cluster
```

### Testing Samples
```bash
make deploy-samples     # Deploy sample CR, configmap, and secret for testing
```

## Linting

```bash
make lint               # Run all linters
make lint-fix           # Run golangci-lint with auto-fix
```

The following linters are configured:
- `golangci-lint` - Go code quality (configured in `.golangci.yml`)
- `checkmake` - Makefile validation
- `hadolint` - Dockerfile linting
- `typos` - Spell checking
- `markdownlint` - Markdown formatting
- `yamllint` - YAML validation
- `shellcheck` - Shell script analysis

## Code Organization

```
certsuite-operator/
├── api/v1alpha1/                    # CRD type definitions
│   ├── certsuiterun_types.go        # CertsuiteRun spec and status
│   ├── certsuiteconsoleplugin_types.go  # Console plugin types
│   ├── *_webhook.go                 # Validation webhooks
│   └── zz_generated.deepcopy.go     # Generated DeepCopy methods
├── cmd/main.go                      # Operator entrypoint
├── internal/controller/             # Controller implementations
│   ├── certsuiterun_controller.go   # Main reconciliation logic
│   ├── certsuiteconsoleplugin_controller.go  # Console plugin controller
│   ├── cnf-cert-job/                # Pod specification builder
│   │   └── cnfcertjob.go            # Functional options for pod creation
│   ├── definitions/                 # Constants and status definitions
│   │   └── definitions.go
│   └── logger/                      # Controller logging
├── certsuite-sidecar/               # Sidecar application
│   └── app/
│       ├── main.go                  # Sidecar entrypoint
│       ├── claim/                   # Claim file parsing
│       └── cnf-cert-suite-report/   # Report generation
├── config/                          # Kustomize configurations
│   ├── crd/                         # CRD manifests
│   ├── default/                     # Default kustomization
│   ├── manager/                     # Controller deployment
│   ├── rbac/                        # RBAC manifests
│   ├── samples/                     # Sample CRs and configs
│   └── webhook/                     # Webhook configurations
├── plugin/                          # OpenShift console plugin
├── bundle/                          # OLM bundle manifests
└── test/                            # Test suites
    ├── e2e/                         # End-to-end tests
    └── utils/                       # Test utilities
```

## Key Dependencies

### Direct Dependencies
- `sigs.k8s.io/controller-runtime` v0.22.1 - Controller framework
- `k8s.io/client-go` v0.34.1 - Kubernetes client
- `k8s.io/api` v0.34.1 - Kubernetes API types
- `k8s.io/apimachinery` v0.34.2 - API machinery utilities
- `github.com/openshift/api` - OpenShift API types (for console plugin)
- `github.com/onsi/ginkgo/v2` v2.27.3 - BDD testing framework
- `github.com/onsi/gomega` v1.38.2 - Matcher library
- `github.com/sirupsen/logrus` v1.9.3 - Logging (sidecar)
- `github.com/go-logr/logr` v1.4.3 - Structured logging interface

### Tool Dependencies
- `controller-gen` v0.19.0 - CRD and RBAC generation
- `kustomize` v5.2.1 - Kubernetes manifest customization
- `operator-sdk` v1.39.1 - Operator scaffolding and OLM tooling
- `opm` - Operator package manager for catalog building

## Development Guidelines

### Go Version
This project uses Go 1.25.5.

### Testing Framework
Tests use Ginkgo/Gomega BDD framework with envtest for controller tests.

### Adding New Fields to CRDs
1. Edit the type definitions in `api/v1alpha1/*_types.go`
2. Run `make manifests` to regenerate CRD manifests
3. Run `make generate` to regenerate DeepCopy methods
4. Update webhooks if validation is needed

### Controller Reconciliation
The `CertsuiteRunReconciler` handles:
- Creating certification pods when a new `CertsuiteRun` CR is detected
- Monitoring pod status and updating CR status phases
- Cleaning up pods when CRs are deleted
- Handling timeouts and error conditions

Status phases: `CertSuiteDeploying` -> `CertSuiteRunning` -> `CertSuiteFinished` (or `CertSuiteError`/`CertSuiteTimeout`)

### Environment Variables
The controller requires:
- `WATCH_NAMESPACE` - Namespace to watch for CRs
- `SIDECAR_APP_IMG` - Sidecar container image to use

### Image Customization
Default images are defined in the Makefile:
- `IMG` - Controller image (default: `quay.io/redhat-best-practices-for-k8s/certsuite-operator:v<VERSION>`)
- `SIDECAR_IMG` - Sidecar image
- `BUNDLE_IMG` - OLM bundle image
- `CATALOG_IMG` - OLM catalog image

Override via environment variables or make arguments:
```bash
make docker-build IMG=myregistry/my-operator:v1.0.0
```

### Local Development with Kind
```bash
# Build local images
scripts/ci/build.sh

# Deploy to kind cluster (preloads images)
scripts/ci/deploy.sh
```

## Common Workflows

### Running a Certification Test
1. Create a ConfigMap with `tnf_config.yaml` content
2. Optionally create a Secret with preflight credentials
3. Create a `CertsuiteRun` CR referencing the ConfigMap and Secret
4. Monitor the CR status for results

### Viewing Test Results
```bash
oc get certsuiteruns -n certsuite-operator
oc get certsuiteruns <name> -o jsonpath='{.status.report.verdict}'
oc get certsuiteruns <name> -o json | jq '.status.report.results'
```

### Debugging
- Check controller logs: `oc logs -n certsuite-operator deployment/certsuite-controller-manager`
- Check certification pod logs: `oc logs -n certsuite-operator certsuite-job-run-<N> -c certsuite`
- Check sidecar logs: `oc logs -n certsuite-operator certsuite-job-run-<N> -c certsuite-sidecar`
