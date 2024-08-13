package claim

import (
	corev1 "k8s.io/api/core/v1"
)

const (
	TestCaseResultPassed  = "passed"
	TestCaseResultSkipped = "skipped"
	TestCaseResultFailed  = "failed"
)

type Metadata struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

type Resource struct {
	Metadata Metadata `json:"metadata,omitempty"`
}

type TestSuiteConfiguration struct {
	NameSpaces        []string     `json:"testNamespaces,omitempty"`
	Pods              []corev1.Pod `json:"testPods,omitempty"`
	Deployments       []Resource   `json:"testDeployments,omitempty"`
	StatefulSets      []Resource   `json:"testStatefulSets,omitempty"`
	Csvs              []Metadata   `json:"AllOperators,omitempty"`
	Crds              []Resource   `json:"testCrds,omitempty"`
	Services          []Resource   `json:"testServices,omitempty"`
	HelmChartReleases []Resource   `json:"testHelmChartReleases,omitempty"`
}

type TestSuiteNodes struct {
	NodeSummary map[string]interface{} `json:"nodeSummary,omitempty"`
}

type TestCaseResult struct {
	CapturedTestOutput string `json:"capturedTestOutput"`
	CatalogInfo        struct {
		BestPracticeReference string `json:"bestPracticeReference"`
		Description           string `json:"description"`
		ExceptionProcess      string `json:"exceptionProcess"`
		Remediation           string `json:"remediation"`
	} `json:"catalogInfo"`
	CategoryClassification map[string]string `json:"categoryClassification"`
	CheckDetails           string            `json:"checkDetails"`
	Duration               int               `json:"duration"`
	EndTime                string            `json:"endTime"`
	FailureLineContent     string            `json:"failureLineContent"`
	FailureLocation        string            `json:"failureLocation"`
	FailureReason          string            `json:"failureReason"`
	SkipReason             string            `json:"skipReason"`
	StartTime              string            `json:"startTime"`
	State                  string            `json:"state"`
	TestID                 struct {
		ID    string `json:"id"`
		Suite string `json:"suite"`
		Tags  string `json:"tags"`
	} `json:"testID"`
}

// Maps a test suite name to a list of TestCaseResult
type TestSuiteResults map[string]TestCaseResult

type Schema struct {
	Claim struct {
		Configurations TestSuiteConfiguration `json:"configurations"`
		Nodes          TestSuiteNodes         `json:"nodes"`
		Results        TestSuiteResults       `json:"results"`
		Versions       struct {
			ClaimFormat string `json:"claimFormat"`
			Ocp         string `json:"ocp"`
			Tnf         string `json:"tnf"`
		} `json:"versions"`
	} `json:"claim"`
}
