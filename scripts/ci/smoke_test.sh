#!/bin/bash
#
# This is a very simple functional test the controller and the expected
# CertsuiteReport CR that should be generated.
#
# Applies the CnfCertificatioSuiteRun sample in config/samples along with its
# related configmap and (preflight) secret. The script waits for the field
# status.phase of the Run CR to be "CertSuiteFinished". Then, it checks the
# report.summary values to match the expected ones.
#
# A patch to the kustomize.yaml file in config/samples needs to be done to add
# the configmap and the secret.
#

# Bash settings: display (expanded) commands and fast exit on first error.
set -o xtrace
set -o errexit

DEFAULT_CNF_CERTSUITE_OPERATOR_NAMESPACE="certsuite-operator"
CNF_CERTSUITE_OPERATOR_NAMESPACE=${CNF_CERTSUITE_OPERATOR_NAMESPACE:-$DEFAULT_CNF_CERTSUITE_OPERATOR_NAMESPACE}

# Checks every $2 secs that .status.phase is $1.
# $3: timeout mins
checkStatusPhase() {
    local polling_interval_secs=$1
    local expected_phase=$2
    local duration_mins=$3

    echo "Checking phase.status to be ${expected_phase}, for ${duration_mins} minutes. Check interval time ${polling_interval_secs} secs."
    runtime="${duration_mins} minute"
    endtime=$(date -ud "$runtime" +%s)

    while [[ $(date -u +%s) -le $endtime ]]; do
        actual_phase=$(oc get certsuiteruns -n certsuite-operator certsuiterun-sample -o json | jq -r '.status.phase')
        if [ "$actual_phase" == "$expected_phase" ]; then
            echo "Phase ${expected_phase} found!"
            return 0
        fi
        echo "Phase ${actual_phase} doesn't match the expected ${expected_phase}. Waiting ${polling_interval_secs} secs..."
        sleep "${polling_interval_secs}"
    done

    return 1
}

# Load samples, patching the kustomization.yaml to include the configmap and the preflight dockerconfig.
make deploy-samples

# Wait 2mins for the CertsuiteRun CR's status.phase to be CertSuiteFinished.
checkStatusPhase 5 CertSuiteFinished 2

# Disable command expansion to avoid output mangling.
set +o xtrace

# Save the Run CR in JSON.
crJson=$(oc get certsuiterun -n "${CNF_CERTSUITE_OPERATOR_NAMESPACE}" certsuiterun-sample -o json)

# Show the Run CR for debugging purposes.
echo "$crJson" | jq

# Run checks for verdict and counters.
export EXPECTED_VERDICT=${EXPECTED_VERDICT:-"pass"}
export EXPECTED_TOTAL_TCS=${EXPECTED_TOTAL_TCS:-"101"}
export EXPECTED_FAILED=${EXPECTED_FAILED:-"0"}
export EXPECTED_PASSED=${EXPECTED_PASSED:-"1"}
export EXPECTED_SKIPPED=${EXPECTED_SKIPPED:-"100"}

# Check the verdit is pass
echo "$crJson" | jq 'if .status.report.verdict == env.EXPECTED_VERDICT then "verdict is "+env.EXPECTED_VERDICT else error("verdict mismatch: \(.status.report.verdict), expected "+env.EXPECTED_VERDICT) end'
echo "$crJson" | jq 'if .status.report.summary.total   | tostring == env.EXPECTED_TOTAL_TCS then "total tc num is ok"   else error("total tcs mismatch: \(.status.report.summary.total), expected "+env.EXPECTED_TOTAL_TCS) end'
echo "$crJson" | jq 'if .status.report.summary.passed  | tostring == env.EXPECTED_PASSED    then "passed tc num is ok"  else error("passed tcs mismatch: \(.status.report.summary.passed), expected "+env.EXPECTED_PASSED) end'
echo "$crJson" | jq 'if .status.report.summary.skipped | tostring == env.EXPECTED_SKIPPED   then "skipped tc num is ok" else error("skipped tcs mismatch: \(.status.report.summary.skipped), expected "+env.EXPECTED_SKIPPED) end'
echo "$crJson" | jq 'if .status.report.summary.failed  | tostring == env.EXPECTED_FAILED    then "failed tc num is ok" else error("failed tcs mismatch: \(.status.report.summary.failed), expected "+env.EXPECTED_FAILED) end'
