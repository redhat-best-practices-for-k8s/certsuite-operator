package cnfcertsuitereport

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cnfcertificationsv1alpha1 "github.com/redhat-best-practices-for-k8s/certsuite-operator/api/v1alpha1"
	"github.com/redhat-best-practices-for-k8s/certsuite-operator/certsuite-sidecar/app/claim"
)

func makeClaimSchema(results claim.TestSuiteResults) *claim.Schema {
	s := &claim.Schema{}
	s.Claim.Results = results
	s.Claim.Versions.Ocp = "4.14.0"
	s.Claim.Versions.Tnf = "5.0.0"
	return s
}

func makeRunCR() *cnfcertificationsv1alpha1.CertsuiteRun {
	return &cnfcertificationsv1alpha1.CertsuiteRun{
		ObjectMeta: metav1.ObjectMeta{Name: "test-run", Namespace: "test-ns"},
	}
}

func TestSetRunCRStatus_Verdicts(t *testing.T) {
	tests := []struct {
		name            string
		results         claim.TestSuiteResults
		expectedVerdict string
		expectedSummary cnfcertificationsv1alpha1.CertsuiteReportStatusSummary
	}{
		{
			name: "all passed -> pass verdict",
			results: claim.TestSuiteResults{
				"test-1": {State: cnfcertificationsv1alpha1.StatusStatePassed},
				"test-2": {State: cnfcertificationsv1alpha1.StatusStatePassed},
			},
			expectedVerdict: cnfcertificationsv1alpha1.StatusVerdictPass,
			expectedSummary: cnfcertificationsv1alpha1.CertsuiteReportStatusSummary{
				Total: 2, Passed: 2,
			},
		},
		{
			name: "one failed -> fail verdict",
			results: claim.TestSuiteResults{
				"test-1": {State: cnfcertificationsv1alpha1.StatusStatePassed},
				"test-2": {State: cnfcertificationsv1alpha1.StatusStateFailed, FailureReason: "bad config"},
			},
			expectedVerdict: cnfcertificationsv1alpha1.StatusVerdictFail,
			expectedSummary: cnfcertificationsv1alpha1.CertsuiteReportStatusSummary{
				Total: 2, Passed: 1, Failed: 1,
			},
		},
		{
			name: "one errored -> error verdict (takes priority over fail)",
			results: claim.TestSuiteResults{
				"test-1": {State: cnfcertificationsv1alpha1.StatusStateFailed},
				"test-2": {State: cnfcertificationsv1alpha1.StatusStateError},
			},
			expectedVerdict: cnfcertificationsv1alpha1.StatusVerdictError,
			expectedSummary: cnfcertificationsv1alpha1.CertsuiteReportStatusSummary{
				Total: 2, Failed: 1, Errored: 1,
			},
		},
		{
			name: "all skipped -> skip verdict",
			results: claim.TestSuiteResults{
				"test-1": {State: cnfcertificationsv1alpha1.StatusStateSkipped, SkipReason: "n/a"},
				"test-2": {State: cnfcertificationsv1alpha1.StatusStateSkipped, SkipReason: "n/a"},
			},
			expectedVerdict: cnfcertificationsv1alpha1.StatusVerdictSkip,
			expectedSummary: cnfcertificationsv1alpha1.CertsuiteReportStatusSummary{
				Total: 2, Skipped: 2,
			},
		},
		{
			name:            "empty results -> skip verdict (0 == 0 skipped)",
			results:         claim.TestSuiteResults{},
			expectedVerdict: cnfcertificationsv1alpha1.StatusVerdictSkip,
			expectedSummary: cnfcertificationsv1alpha1.CertsuiteReportStatusSummary{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runCR := makeRunCR()
			claimSchema := makeClaimSchema(tc.results)
			SetRunCRStatus(runCR, claimSchema)

			require.NotNil(t, runCR.Status.Report)
			assert.Equal(t, tc.expectedVerdict, runCR.Status.Report.Verdict)
			assert.Equal(t, tc.expectedSummary, runCR.Status.Report.Summary)
			assert.Equal(t, "4.14.0", runCR.Status.Report.OcpVersion)
			assert.Equal(t, "5.0.0", runCR.Status.Report.CnfCertSuiteVersion)
		})
	}
}

func TestSetRunCRStatus_PerStateFields(t *testing.T) {
	t.Run("passed test has no reason/remediation", func(t *testing.T) {
		runCR := makeRunCR()
		claimSchema := makeClaimSchema(claim.TestSuiteResults{
			"test-pass": {
				State:              cnfcertificationsv1alpha1.StatusStatePassed,
				CapturedTestOutput: "some logs",
			},
		})
		SetRunCRStatus(runCR, claimSchema)
		require.Len(t, runCR.Status.Report.Results, 1)
		r := runCR.Status.Report.Results[0]
		assert.Equal(t, cnfcertificationsv1alpha1.StatusStatePassed, r.Result)
		assert.Empty(t, r.Reason)
		assert.Empty(t, r.Remediation)
		assert.Empty(t, r.Logs)
	})

	t.Run("skipped test populates reason from SkipReason", func(t *testing.T) {
		runCR := makeRunCR()
		claimSchema := makeClaimSchema(claim.TestSuiteResults{
			"test-skip": {
				State:      cnfcertificationsv1alpha1.StatusStateSkipped,
				SkipReason: "not applicable",
			},
		})
		SetRunCRStatus(runCR, claimSchema)
		require.Len(t, runCR.Status.Report.Results, 1)
		assert.Equal(t, "not applicable", runCR.Status.Report.Results[0].Reason)
	})

	t.Run("failed test populates reason, logs, and remediation", func(t *testing.T) {
		runCR := makeRunCR()
		claimSchema := makeClaimSchema(claim.TestSuiteResults{
			"test-fail": {
				State:              cnfcertificationsv1alpha1.StatusStateFailed,
				FailureReason:      "pod missing label",
				CapturedTestOutput: "debug output",
				CatalogInfo: struct {
					BestPracticeReference string `json:"bestPracticeReference"`
					Description           string `json:"description"`
					ExceptionProcess      string `json:"exceptionProcess"`
					Remediation           string `json:"remediation"`
				}{Remediation: "add the label"},
			},
		})
		SetRunCRStatus(runCR, claimSchema)
		require.Len(t, runCR.Status.Report.Results, 1)
		r := runCR.Status.Report.Results[0]
		assert.Equal(t, "pod missing label", r.Reason)
		assert.Equal(t, "debug output", r.Logs)
		assert.Equal(t, "add the label", r.Remediation)
	})
}

func TestSetRunCRStatus_ShowAllResultsLogs(t *testing.T) {
	runCR := makeRunCR()
	runCR.Spec.ShowAllResultsLogs = true
	claimSchema := makeClaimSchema(claim.TestSuiteResults{
		"test-pass": {
			State:              cnfcertificationsv1alpha1.StatusStatePassed,
			CapturedTestOutput: "some output",
		},
	})
	SetRunCRStatus(runCR, claimSchema)
	require.Len(t, runCR.Status.Report.Results, 1)
	assert.Equal(t, "some output", runCR.Status.Report.Results[0].Logs)
}

func TestSetRunCRStatus_ShowCompliantResourcesAlways(t *testing.T) {
	checkDetails := CheckDetails{
		Compliant: []map[string]interface{}{
			{"ObjectFieldsKeys": []interface{}{"name"}, "ObjectFieldsValues": []interface{}{"pod-1"}},
		},
	}
	checkDetailsJSON, _ := json.Marshal(checkDetails)

	runCR := makeRunCR()
	runCR.Spec.ShowCompliantResourcesAlways = true
	claimSchema := makeClaimSchema(claim.TestSuiteResults{
		"test-pass": {
			State:        cnfcertificationsv1alpha1.StatusStatePassed,
			CheckDetails: string(checkDetailsJSON),
		},
	})
	SetRunCRStatus(runCR, claimSchema)
	require.Len(t, runCR.Status.Report.Results, 1)
	require.NotNil(t, runCR.Status.Report.Results[0].TargetResources)
	assert.Len(t, runCR.Status.Report.Results[0].TargetResources.Compliant, 1)
}

func TestSetTestCaseTargets(t *testing.T) {
	t.Run("empty checkDetails is no-op", func(t *testing.T) {
		result := &cnfcertificationsv1alpha1.TestCaseResult{}
		setTestCaseTargets("tc1", "", result)
		assert.Nil(t, result.TargetResources)
	})

	t.Run("invalid JSON is no-op", func(t *testing.T) {
		result := &cnfcertificationsv1alpha1.TestCaseResult{}
		setTestCaseTargets("tc1", "not-json", result)
		assert.Nil(t, result.TargetResources)
	})

	t.Run("valid JSON populates target resources", func(t *testing.T) {
		details := CheckDetails{
			Compliant: []map[string]interface{}{
				{"ObjectFieldsKeys": []interface{}{"ns", "name"}, "ObjectFieldsValues": []interface{}{"default", "pod-a"}},
			},
			NonCompliant: []map[string]interface{}{
				{"ObjectFieldsKeys": []interface{}{"ns"}, "ObjectFieldsValues": []interface{}{"bad-ns"}},
			},
		}
		detailsJSON, _ := json.Marshal(details)
		result := &cnfcertificationsv1alpha1.TestCaseResult{}
		setTestCaseTargets("tc1", string(detailsJSON), result)

		require.NotNil(t, result.TargetResources)
		require.Len(t, result.TargetResources.Compliant, 1)
		assert.Equal(t, "default", result.TargetResources.Compliant[0]["ns"])
		assert.Equal(t, "pod-a", result.TargetResources.Compliant[0]["name"])
		require.Len(t, result.TargetResources.NonCompliant, 1)
		assert.Equal(t, "bad-ns", result.TargetResources.NonCompliant[0]["ns"])
	})
}

func TestGetTargetResourcesFromClaim(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		result := getTargetResourcesFromClaim(nil)
		assert.Nil(t, result)
	})

	t.Run("empty input returns nil", func(t *testing.T) {
		result := getTargetResourcesFromClaim([]map[string]interface{}{})
		assert.Nil(t, result)
	})

	t.Run("multiple targets with multiple fields", func(t *testing.T) {
		input := []map[string]interface{}{
			{
				"ObjectFieldsKeys":   []interface{}{"ns", "name"},
				"ObjectFieldsValues": []interface{}{"default", "pod-1"},
			},
			{
				"ObjectFieldsKeys":   []interface{}{"kind"},
				"ObjectFieldsValues": []interface{}{"Pod"},
			},
		}
		result := getTargetResourcesFromClaim(input)
		require.Len(t, result, 2)
		assert.Equal(t, "default", result[0]["ns"])
		assert.Equal(t, "pod-1", result[0]["name"])
		assert.Equal(t, "Pod", result[1]["kind"])
	})
}

func TestAddNamespacesToCnfSpecField(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		cnf := &cnfcertificationsv1alpha1.CnfTargets{}
		addNamespacesToCnfSpecField(cnf, nil)
		assert.Nil(t, cnf.Namespaces)
	})

	t.Run("appends namespaces", func(t *testing.T) {
		cnf := &cnfcertificationsv1alpha1.CnfTargets{}
		addNamespacesToCnfSpecField(cnf, []string{"ns-a", "ns-b"})
		assert.Equal(t, []string{"ns-a", "ns-b"}, cnf.Namespaces)
	})

	t.Run("appends to existing", func(t *testing.T) {
		cnf := &cnfcertificationsv1alpha1.CnfTargets{Namespaces: []string{"existing"}}
		addNamespacesToCnfSpecField(cnf, []string{"new"})
		assert.Equal(t, []string{"existing", "new"}, cnf.Namespaces)
	})
}

func TestAddPodsToCnfSpecField(t *testing.T) {
	t.Run("empty pods", func(t *testing.T) {
		cnf := &cnfcertificationsv1alpha1.CnfTargets{}
		addPodsToCnfSpecField(cnf, nil)
		assert.Nil(t, cnf.Pods)
	})

	t.Run("pod with containers", func(t *testing.T) {
		cnf := &cnfcertificationsv1alpha1.CnfTargets{}
		pods := []corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "pod-1", Namespace: "ns-1"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "app"},
						{Name: "sidecar"},
					},
				},
			},
		}
		addPodsToCnfSpecField(cnf, pods)
		require.Len(t, cnf.Pods, 1)
		assert.Equal(t, "pod-1", cnf.Pods[0].Name)
		assert.Equal(t, "ns-1", cnf.Pods[0].Namespace)
		assert.Equal(t, []string{"app", "sidecar"}, cnf.Pods[0].Containers)
	})

	t.Run("pod with zero containers", func(t *testing.T) {
		cnf := &cnfcertificationsv1alpha1.CnfTargets{}
		pods := []corev1.Pod{
			{ObjectMeta: metav1.ObjectMeta{Name: "empty-pod"}},
		}
		addPodsToCnfSpecField(cnf, pods)
		require.Len(t, cnf.Pods, 1)
		assert.Nil(t, cnf.Pods[0].Containers)
	})
}

func TestAddOperatorsToCnfSpecField(t *testing.T) {
	cnf := &cnfcertificationsv1alpha1.CnfTargets{}
	csvs := []claim.Metadata{
		{Name: "operator-a", Namespace: "ns-1"},
		{Name: "operator-b", Namespace: "ns-2"},
	}
	addOperatorsToCnfSpecField(cnf, csvs)
	require.Len(t, cnf.Csvs, 2)
	assert.Equal(t, "operator-a", cnf.Csvs[0].Name)
	assert.Equal(t, "ns-2", cnf.Csvs[1].Namespace)
}

func TestAddCrdsToCnfSpecField(t *testing.T) {
	cnf := &cnfcertificationsv1alpha1.CnfTargets{}
	crds := []claim.Resource{
		{Metadata: claim.Metadata{Name: "crd-a.example.com"}},
		{Metadata: claim.Metadata{Name: "crd-b.example.com"}},
	}
	addCrdsToCnfSpecField(cnf, crds)
	assert.Equal(t, []string{"crd-a.example.com", "crd-b.example.com"}, cnf.Crds)
}

func TestAddNodesToCnfSpecField(t *testing.T) {
	t.Run("nil map", func(t *testing.T) {
		cnf := &cnfcertificationsv1alpha1.CnfTargets{}
		addNodesToCnfSpecField(cnf, nil)
		assert.Nil(t, cnf.Nodes)
	})

	t.Run("populates node names from map keys", func(t *testing.T) {
		cnf := &cnfcertificationsv1alpha1.CnfTargets{}
		nodes := map[string]interface{}{"node-1": nil, "node-2": nil}
		addNodesToCnfSpecField(cnf, nodes)
		assert.Len(t, cnf.Nodes, 2)
		assert.Contains(t, cnf.Nodes, "node-1")
		assert.Contains(t, cnf.Nodes, "node-2")
	})
}

func TestAddResourcesToCnfSpecField(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		var field []cnfcertificationsv1alpha1.CnfResource
		addResourcesToCnfSpecField(&field, nil)
		assert.Nil(t, field)
	})

	t.Run("converts resources", func(t *testing.T) {
		var field []cnfcertificationsv1alpha1.CnfResource
		resources := []claim.Resource{
			{Metadata: claim.Metadata{Name: "dep-1", Namespace: "ns-1"}},
		}
		addResourcesToCnfSpecField(&field, resources)
		require.Len(t, field, 1)
		assert.Equal(t, "dep-1", field[0].Name)
		assert.Equal(t, "ns-1", field[0].Namespace)
	})
}

func TestGetCnfTargetsFromClaim(t *testing.T) {
	claimSchema := &claim.Schema{}
	claimSchema.Claim.Configurations.NameSpaces = []string{"ns-a"}
	claimSchema.Claim.Nodes.NodeSummary = map[string]interface{}{"node-1": nil}
	claimSchema.Claim.Configurations.Pods = []corev1.Pod{
		{ObjectMeta: metav1.ObjectMeta{Name: "pod-1", Namespace: "ns-a"}},
	}
	claimSchema.Claim.Configurations.Csvs = []claim.Metadata{
		{Name: "csv-1", Namespace: "ns-a"},
	}
	claimSchema.Claim.Configurations.Crds = []claim.Resource{
		{Metadata: claim.Metadata{Name: "crd.example.com"}},
	}

	cnf := getCnfTargetsFromClaim(claimSchema)
	assert.Equal(t, []string{"ns-a"}, cnf.Namespaces)
	assert.Contains(t, cnf.Nodes, "node-1")
	require.Len(t, cnf.Pods, 1)
	assert.Equal(t, "pod-1", cnf.Pods[0].Name)
	require.Len(t, cnf.Csvs, 1)
	assert.Equal(t, "csv-1", cnf.Csvs[0].Name)
	assert.Equal(t, []string{"crd.example.com"}, cnf.Crds)
}

func TestNew(t *testing.T) {
	config := &Config{
		OcpVersion:          "4.14.0",
		CnfCertSuiteVersion: "5.0.0",
		Cnf: cnfcertificationsv1alpha1.CnfTargets{
			Namespaces: []string{"ns-1"},
		},
	}
	report := New(config)
	assert.Equal(t, "4.14.0", report.OcpVersion)
	assert.Equal(t, "5.0.0", report.CnfCertSuiteVersion)
	assert.Equal(t, []string{"ns-1"}, report.CnfTargets.Namespaces)
}
