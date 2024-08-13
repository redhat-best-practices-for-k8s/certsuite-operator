package cnfcertsuitereport

import (
	"encoding/json"

	cnfcertificationsv1alpha1 "github.com/redhat-best-practices-for-k8s/certsuite-operator/api/v1alpha1"
	"github.com/redhat-best-practices-for-k8s/certsuite-operator/certsuite-sidecar/app/claim"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

type Config struct {
	ReportCrName           string
	Namespace              string
	CertSuiteConfigRunName string
	OcpVersion             string
	CnfCertSuiteVersion    string
	Cnf                    cnfcertificationsv1alpha1.CnfTargets
}

type CheckDetails struct {
	Compliant    []map[string]interface{} `json:"CompliantObjectsOut"`
	NonCompliant []map[string]interface{} `json:"NonCompliantObjectsOut"`
}

func addNamespacesToCnfSpecField(cnf *cnfcertificationsv1alpha1.CnfTargets, namespaces []string) {
	cnf.Namespaces = append(cnf.Namespaces, namespaces...)
}

func addPodsToCnfSpecField(cnf *cnfcertificationsv1alpha1.CnfTargets, pods []corev1.Pod) {
	for podIdx := 0; podIdx < len(pods); podIdx++ {
		pod := &pods[podIdx]
		podsContainers := &pod.Spec.Containers
		var containers []string
		for containerIdx := 0; containerIdx < len(*podsContainers); containerIdx++ {
			containers = append(containers, (*podsContainers)[containerIdx].Name)
		}
		cnf.Pods = append(cnf.Pods, cnfcertificationsv1alpha1.CnfPod{
			Name:       pod.Name,
			Namespace:  pod.Namespace,
			Containers: containers,
		})
	}
}

func addOperatorsToCnfSpecField(cnf *cnfcertificationsv1alpha1.CnfTargets, csvs []claim.Metadata) {
	for _, csv := range csvs {
		cnf.Csvs = append(cnf.Csvs, cnfcertificationsv1alpha1.CnfResource{
			Name:      csv.Name,
			Namespace: csv.Namespace,
		})
	}
}

func addCrdsToCnfSpecField(cnf *cnfcertificationsv1alpha1.CnfTargets, crds []claim.Resource) {
	for _, crd := range crds {
		cnf.Crds = append(cnf.Crds, crd.Metadata.Name)
	}
}

func addNodesToCnfSpecField(cnf *cnfcertificationsv1alpha1.CnfTargets, nodes map[string]interface{}) {
	for nodeName := range nodes {
		cnf.Nodes = append(cnf.Nodes, nodeName)
	}
}

func addResourcesToCnfSpecField(cnfResourceField *[]cnfcertificationsv1alpha1.CnfResource, resources []claim.Resource) {
	for _, resource := range resources {
		*cnfResourceField = append(*cnfResourceField, cnfcertificationsv1alpha1.CnfResource{
			Name:      resource.Metadata.Name,
			Namespace: resource.Metadata.Namespace,
		})
	}
}

func getCnfTargetsFromClaim(claimcontent *claim.Schema) *cnfcertificationsv1alpha1.CnfTargets {
	var cnf cnfcertificationsv1alpha1.CnfTargets
	addNamespacesToCnfSpecField(&cnf, claimcontent.Claim.Configurations.NameSpaces)
	addNodesToCnfSpecField(&cnf, claimcontent.Claim.Nodes.NodeSummary)
	addPodsToCnfSpecField(&cnf, claimcontent.Claim.Configurations.Pods)
	addResourcesToCnfSpecField(&cnf.Deployments, claimcontent.Claim.Configurations.Deployments)
	addResourcesToCnfSpecField(&cnf.StatefulSets, claimcontent.Claim.Configurations.StatefulSets)
	addOperatorsToCnfSpecField(&cnf, claimcontent.Claim.Configurations.Csvs)
	addCrdsToCnfSpecField(&cnf, claimcontent.Claim.Configurations.Crds)
	addResourcesToCnfSpecField(&cnf.Services, claimcontent.Claim.Configurations.Services)
	addResourcesToCnfSpecField(&cnf.HelmChartReleases, claimcontent.Claim.Configurations.HelmChartReleases)
	return &cnf
}

func New(config *Config) *cnfcertificationsv1alpha1.CnfCertificationSuiteReport {
	return &cnfcertificationsv1alpha1.CnfCertificationSuiteReport{
		OcpVersion:          config.OcpVersion,
		CnfCertSuiteVersion: config.CnfCertSuiteVersion,
		CnfTargets:          config.Cnf,
	}
}

func SetRunCRStatus(runCR *cnfcertificationsv1alpha1.CnfCertificationSuiteRun, claimSchema *claim.Schema) {
	testSuiteResults := &claimSchema.Claim.Results
	results := []cnfcertificationsv1alpha1.TestCaseResult{}
	totalTests, passedTests, skippedTests, failedTests, erroredTests := 0, 0, 0, 0, 0
	for tcName := range *testSuiteResults {
		tcResult := (*testSuiteResults)[tcName]
		testCaseResult := cnfcertificationsv1alpha1.TestCaseResult{
			TestCaseName: tcName,
			Result:       tcResult.State,
		}

		switch tcResult.State {
		case cnfcertificationsv1alpha1.StatusStatePassed:
			passedTests++
		case cnfcertificationsv1alpha1.StatusStateSkipped:
			skippedTests++
			testCaseResult.Reason = tcResult.SkipReason
		case cnfcertificationsv1alpha1.StatusStateFailed:
			failedTests++
			testCaseResult.Reason = tcResult.FailureReason
			testCaseResult.Logs = tcResult.CapturedTestOutput
		case cnfcertificationsv1alpha1.StatusStateError:
			erroredTests++
		}

		// Always show failed compliant and non-compliant resources
		if tcResult.State == cnfcertificationsv1alpha1.StatusStateFailed ||
			(tcResult.State == cnfcertificationsv1alpha1.StatusStatePassed && runCR.Spec.ShowCompliantResourcesAlways) {
			setTestCaseTargets(tcName, tcResult.CheckDetails, &testCaseResult)
		}
		if runCR.Spec.ShowAllResultsLogs {
			testCaseResult.Logs = tcResult.CapturedTestOutput
		}
		totalTests++
		results = append(results, testCaseResult)
	}

	runCR.Status.Report = &cnfcertificationsv1alpha1.CnfCertificationSuiteReport{
		OcpVersion:          claimSchema.Claim.Versions.Ocp,
		CnfCertSuiteVersion: claimSchema.Claim.Versions.Tnf,
		CnfTargets:          *getCnfTargetsFromClaim(claimSchema),
		Results:             results,
		Summary: cnfcertificationsv1alpha1.CnfCertificationSuiteReportStatusSummary{
			Total:   totalTests,
			Passed:  passedTests,
			Skipped: skippedTests,
			Failed:  failedTests,
			Errored: erroredTests,
		},
	}

	switch {
	case erroredTests >= 1: // at least one test encountered an error
		runCR.Status.Report.Verdict = cnfcertificationsv1alpha1.StatusVerdictError
	case failedTests >= 1: // at least one failed test
		runCR.Status.Report.Verdict = cnfcertificationsv1alpha1.StatusVerdictFail
	case skippedTests == totalTests: // all tests were skipped
		runCR.Status.Report.Verdict = cnfcertificationsv1alpha1.StatusVerdictSkip
	default: // all tests who ran have passed
		runCR.Status.Report.Verdict = cnfcertificationsv1alpha1.StatusVerdictPass
	}
}

func setTestCaseTargets(tcName, checkDetailsStr string, testCaseResult *cnfcertificationsv1alpha1.TestCaseResult) {
	if checkDetailsStr == "" {
		logrus.Warnf("tcs %s with state %s has an empty checkDetails field", tcName, testCaseResult.Result)
		return
	}
	checkDetails := CheckDetails{}
	err := json.Unmarshal([]byte(checkDetailsStr), &checkDetails)
	if err != nil {
		logrus.Errorf("error unmarsling check details of tc %s: %s", tcName, err)
		return
	}

	testCaseResult.TargetResources = &cnfcertificationsv1alpha1.TargetResources{
		Compliant:    getTargetResourcesFromClaim(checkDetails.Compliant),
		NonCompliant: getTargetResourcesFromClaim(checkDetails.NonCompliant),
	}
}

func getTargetResourcesFromClaim(claimTcTargets []map[string]interface{}) []cnfcertificationsv1alpha1.TargetResource {
	var tcTargets []cnfcertificationsv1alpha1.TargetResource
	for i, c := range claimTcTargets {
		tcTargets = append(tcTargets, map[string]string{})
		objectFieldsKeys := c["ObjectFieldsKeys"].([]interface{})
		objectFieldsValues := c["ObjectFieldsValues"].([]interface{})
		for j, key := range objectFieldsKeys {
			tcTargets[i][key.(string)] = objectFieldsValues[j].(string)
		}
	}
	return tcTargets
}
