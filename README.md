# Certification Suite Operator

[![red hat](https://img.shields.io/badge/red%20hat---?color=gray&logo=redhat&logoColor=red&style=flat)](https://www.redhat.com)
[![openshift](https://img.shields.io/badge/openshift---?color=gray&logo=redhatopenshift&logoColor=red&style=flat)](https://www.redhat.com/en/technologies/cloud-computing/openshift)

## Description

Kubernetes/Openshift Operator (scaffolded with operator-sdk) running the
[Certification Suite Container](https://github.com/redhat-best-practices-for-k8s/certsuite-operator).

The Certification Suites provide a set of test cases for the
Containerized Network Functions/Cloud Native Functions (CNFs) to verify if
best practices for deployment on Red Hat OpenShift clusters are followed.

### How does it work?

The Operator registers a CRD in the cluster: `CnfCertificationSuiteRun`.

In order to fire up the Certification Suite, the user must create
a CnfCertificationSuiteRun CR, also informally referred as Run CR, which
has to be created with a Config Map containing the cnf certification suites configuration,
and a Secret containing the preflight suite credentials.
**Note:** All resources mentioned above should be created in the operator's
installation namespace (by default `certsuite-operator`)

See resources relationship diagram:

![run config](doc/uml/run_config.png)

When the CR is deployed, a new pod with two containers is created:

1. Container built with the cnf certification image in order to run the suites.
2. Container (sidecar) which updates the Run CR's status fields containing the
Certification suites results based on results claim file created by the
previous container.

**See diagram summarizing the process:**

![Use Case Run](doc/uml/use_case_run.png)

## Getting Started

You’ll need a Kubernetes cluster to run against.
You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing,
or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your
kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Install operator

#### Initial steps

Clone Cnf Certification Operator repo:

```sh
git clone https://github.com/redhat-best-practices-for-k8s/certsuite-operator.git
```

#### Option 1: Using OLM subscription

1. Export OLM catalog image and namespace:

    ```sh
    export OLM_INSTALL_IMG_CATALOG=<your-registry.com>/<your-repo>/certsuite-operator-catalog:<version>
    export OLM_INSTALL_NAMESPACE=<your-namespace>
    ```

    **Note:** If the catalog image and namespace aren't provided,
    they will be set by default to:

    ```sh
    OLM_INSTALL_IMG_CATALOG = quay.io/redhat-best-practices-for-k8s/certsuite-operator-catalog:latest
    OLM_INSTALL_NAMESPACE = certsuite-operator
    ```

2. Install Cnf Certification Operator:\
    Use the following make target to install the operator using OLM subscription:

    ```sh
    make olm-install
    ```

3. Verify Installation:
    Run the following command and expect a similar output:

    <!-- markdownlint-disable -->
    ```sh
    $ oc get pods -n $OLM_INSTALL_NAMESPACE
    NAME                                                              READY   STATUS      RESTARTS   AGE
    afa1738b451274ef681c19ae8e8a6dcc50f65568056ef97355a4a2fe14hbhpn   0/1     Completed   0          3m32s
    cnf-certsuite-controller-manager-67f68cd4cb-625ww                 2/2     Running     0          3m18s
    certsuite-operator-olm-catalog-mkmqw                          1/1     Running     0          3m45s
    ```
    <!-- markdownlint-enable -->

    **Note:** If `OLM_INSTALL_NAMESPACE` environment variable wasn't exported
    in previous steps, use `certsuite-operator` as namespace instead.

#### Option 2: Manually building and deploying the operator

In order to Install the operator by manually building and deploying it,
the `cert-manager` has to be installed at first:

```sh
kubectl apply -f https://github.com/jetstack/cert-manager/releases/latest/download/cert-manager.yaml
```

##### Option 2.1: Use your own registry account

1. Export images environment variables:

    ```sh
    export IMG=<your-registry.com>/<your-repo>/certsuite-operator:<version>
    export SIDECAR_IMG=<your-registry.com>/<your-repo>/certsuite-operator-sidecar:<version>
    ```

2. Build and upload the controller image to your registry account:

    ```sh
    make docker-build docker-push
    ```

3. Build and upload the side car image to your registry account:

    ```sh
    docker build -f cnf-cert-sidecar/Dockerfile -t $SIDECAR_IMG .
    docker push $SIDECAR_IMG
    ```

4. Deploy the operator, using the previously uploaded controller image,
 and the built side car image:

    ```sh
    make deploy
    ```

##### Option 2.2: Use local images

1. Export images environment variables (optional):

    ```sh
    export IMG=<your-certsuite-operator-image-name>
    export SIDECAR_IMG=<your-sidecar-app-image-name>
    ```

    **Note**: if the images aren't provided,
    scripts of next steps will use default images:

    ```sh
    IMG=ci-cnf-op:v0.0.1-test
    SIDECAR_IMG=ci-cnf-op-sidecar:v0.0.1-test
    ```

2. Build controller and side car images:

    ```sh
    scripts/ci/build.sh
    ```

3. Deploy previously built images by preloading them into the kind cluster's nodes:

    ```sh
    scripts/ci/deploy.sh
    ```

### Test it out

Use our samples to test out the cnf certification operator, with the following command:

```sh
make deploy-samples
```

**Note**: Current sample CnfCertificationSuiteRun CR configures
the Certification Suite to run the "observability" test suite only.
It can be modified by changing manually the `labelsFilter` of the [sample CR](https://github.com/redhat-best-practices-for-k8s/certsuite-operator/blob/main/config/samples/cnf-certifications_v1alpha1_cnfcertificationsuiterun.yaml).

### How to customize the Certification Suite run

1. Create Resources

    In order to use the cnf certification suite operator,
    you'll have to create yaml files for the following resources:

    1. Config map:\
    Containing the cnf certification configuration file
    content under the `tnf_config.yaml` key.\
    (see [Certification configuration description](https://redhat-best-practices-for-k8s.github.io/certsuite-operator/configuration/))

    2. Secret:\
    Containing cnf preflight suite credentials
    under the `preflight_dockerconfig.json` key.\
    (see [Preflight Integration description](https://redhat-best-practices-for-k8s.github.io/certsuite-operator/runtime-env/#disable-intrusive-tests))

    3. CnfCertificationSuiteRun CR:\
    Containing the following Spec fields that has to be filled in:
        - **labelsFilter**: Wanted label filtering the cnf certification tests suite.
        - **logLevel**: Wanted log level of cnf certification tests suite run.\
        Log level options: "info", "debug", "warn", "error"
        - **timeout**: Wanted timeout for the the cnf certification tests.
        - **configMapName**: Name of the config map defined at stage 1.
        - **preflightSecretName**: Name of the preflight Secret
        defined at stage 2.
        - **enableDataCollection**: Set to "true" to enable data collection,
        or "false" otherwise.\
        **Note:** When setting this field to true, the claim file will be sent to
        a server external to the cluster. see
        [collector repository](https://github.com/redhat-best-practices-for-k8s/collector)
        for more details
        - **showAllResultsLogs**: Set to "true" to show all result's logs,
        and not only logs of failed test cases.
        This field is set to "false" by default.
        - **showCompliantResourcesAlways**: Set to "true" to show compliant
        resources of all results. and not only compliant and non-compliant
        resources of failed test cases. This field is set to "false" by default.

        See a [sample CnfCertificationSuiteRun CR](https://github.com/redhat-best-practices-for-k8s/certsuite-operator/blob/main/config/samples/cnf-certifications_v1alpha1_cnfcertificationsuiterun.yaml)

2. Apply resources into the cluster

    After creating all the yaml files for required resources,
    use the following commands to apply them into the cluster:

    ```sh
    oc apply -f /path/to/config/map.yaml
    oc apply -f /path/to/preflight/secret.yaml
    oc apply -f /path/to/cnfCertificationSuiteRun.yaml
    ```

    **Note**: The same config map and secret can be reused
    by different CnfCertificationSuiteRun CR's.

### Review results

If all of the resources were applied successfully, the cnf certification suites
will run on a new created `pod` in the `certsuite-operator` namespace.
The pod has the name with the form `cnf-job-run-N`:

<!-- markdownlint-disable -->
```sh
$ oc get pods -n certsuite-operator 
NAME                                                READY   STATUS      RESTARTS   AGE
cnf-certsuite-controller-manager-6c6bb6d965-jslmd   2/2     Running     0          21h
cnf-job-run-1                                       0/2     Completed   0          21h
```
<!-- markdownlint-enable -->

Check whether the pod creation and the cnf certification suites run were successful
by checking CnfCertificationSuiteRun CR's status.
In the successful case, expect to see the following status:

```sh
$ oc get cnfcertificationsuiteruns.cnf-certifications.redhat.com -n certsuite-operator
NAME                              AGE   STATUS
cnfcertificationsuiterun-sample   50m   CertSuiteFinished
```

The status `CertSuiteFinished` means the Cert Suite pod has finished running
all the test cases, so the results can be inspected in field `report` of the Run
CR's (cnfcertificationsuiterun-sample) status subresource.

- Results: For every test case, contains its result and logs.
If the the result is "skipped" or "failed" contains also the skip\failure reason.

    See example:

    <!-- markdownlint-disable -->
    ```sh
    status:
       report:
            results:
                - logs: |
                    INFO  [Feb 15 13:05:50.749] [check.go: 263] [observability-pod-disruption-budget] Running check (labels: [common observability-pod-disruption-budget observability])
                    INFO  [Feb 15 13:05:50.749] [suite.go: 193] [observability-pod-disruption-budget] Testing Deployment "deployment: test ns: tnf"
                    INFO  [Feb 15 13:05:50.749] [suite.go: 206] [observability-pod-disruption-budget] PDB "test-pdb-min" is valid for Deployment: "test"
                    INFO  [Feb 15 13:05:50.749] [suite.go: 224] [observability-pod-disruption-budget] Testing StatefulSet "statefulset: test ns: tnf"
                    INFO  [Feb 15 13:05:50.749] [suite.go: 237] [observability-pod-disruption-budget] PDB "test-pdb-max" is valid for StatefulSet: "test"
                    INFO  [Feb 15 13:05:50.749] [checksdb.go: 115] [observability-pod-disruption-budget] Recording result "PASSED", claimID: {Id:observability-pod-disruption-budget Suite:observability Tags:common}
                    result: passed
                    testCaseName: observability-pod-disruption-budget
                - logs: |
                    INFO  [Feb 15 13:05:50.723] [checksgroup.go: 83] [operator-install-source] Skipping check operator-install-source, reason: no matching labels
                    INFO  [Feb 15 13:05:50.723] [checksdb.go: 115] [operator-install-source] Recording result "SKIPPED", claimID: {Id:operator-install-source Suite:operator Tags:common}
                    reason: no matching labels
                    result: skipped
                    testCaseName: operator-install-source
                - logs: |
                    INFO  [Feb 15 13:05:50.749] [checksgroup.go: 83] [affiliated-certification-helmchart-is-certified] Skipping check affiliated-certification-helmchart-is-certified, reason: no matching labels
                    INFO  [Feb 15 13:05:50.749] [checksdb.go: 115] [affiliated-certification-helmchart-is-certified] Recording result "SKIPPED", claimID: {Id:affiliated-certification-helmchart-is-certified Suite:affiliated-certification Tags:common}
                    reason: no matching labels
                    result: skipped
                    testCaseName: affiliated-certification-helmchart-is-certified
    ```
    <!-- markdownlint-enable -->

- Summary: Summarize the total number of tests by their results.
- Verdict: Specifies the overall result of the certificattion suites run.\
Poissible verdicts: "pass", "skip", "fail", "error".

Run the following command to ensure its creation:

<!-- markdownlint-disable -->
```sh
$ oc get cnfcertificationsuiteruns.cnf-certifications.redhat.com -n certsuite-operator cnfcertificationsuiterun-sample -o json | jq '.status.report.verdict'
"pass"
```
<!-- markdownlint-enable -->

### Uninstall CRDs

To delete the CRDs from the cluster:

```sh
make uninstall
```

### Undeploy controller

#### Option 1: Operator was installed using OLM subscription

Uninstall operator:

```sh
make olm-uninstall
```

#### Option 2: Operator was Manually built and deployed

Undeploy the controller from the cluster:

```sh
make undeploy
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

```plaintext
http://www.apache.org/licenses/LICENSE-2.0
```

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
