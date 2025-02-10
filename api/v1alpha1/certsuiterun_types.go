/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CertsuiteRunSpec defines the desired state of CertsuiteRun
type CertsuiteRunSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// LabelsFilter holds the labels filter/expression of the test cases we want to run.
	LabelsFilter string `json:"labelsFilter"`
	// LogLevel sets the CNF Certification Suite log level (TNF_LOG_LEVEL)
	LogLevel string `json:"logLevel"`

	// Total timeout for the CNF Cert Suite to run.
	TimeOut string `json:"timeout"`
	// ConfigMapName holds the cnf certification suite yaml config.
	ConfigMapName string `json:"configMapName"`
	// PreflightSecretName holds the secret name for preflight's dockerconfig.
	PreflightSecretName *string `json:"preflightSecretName,omitempty"`

	// EnableDataCollection is set to true to enable sending results claim file to the "Collector" app, for storing its data.
	EnableDataCollection bool `json:"enableDataCollection,omitempty"`
	// ShowAllResultsLogs is set to true for showing all test results logs, and not only of failed tcs.
	ShowAllResultsLogs bool `json:"showAllResultsLogs,omitempty"`
	// ShowCompliantResourcesAlways is set true for showing compliant resources for all ran tcs, and not only of failed tcs.
	ShowCompliantResourcesAlways bool `json:"showCompliantResourcesAlways,omitempty"`
}

type StatusPhase string

const (
	StatusPhaseCertSuiteDeploying   = "CertSuiteDeploying"
	StatusPhaseCertSuiteDeployError = "CertSuiteDeployError"
	StatusPhaseCertSuiteRunning     = "CertSuiteRunning"
	StatusPhaseCertSuiteFinished    = "CertSuiteFinished"
	StatusPhaseCertSuiteError       = "CertSuiteError"
)

const (
	StatusStatePassed  = "passed"
	StatusStateSkipped = "skipped"
	StatusStateFailed  = "failed"
	StatusStateError   = "error"
)

const (
	StatusVerdictPass  = "pass"
	StatusVerdictSkip  = "skip"
	StatusVerdictFail  = "fail"
	StatusVerdictError = "error"
)

// CertsuiteRunStatus defines the observed state of CertsuiteRun
type CertsuiteRunStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Phase holds the current phase of the CNF Certification Suite run.
	//+kubebuilder:validation:Enum=CertSuiteDeploying;CertSuiteDeployFailure;CertSuiteRunning;CertSuiteFinished;CertSuiteError
	Phase StatusPhase `json:"phase"`
	// CnfCertSuitePodName holds the name of the pod where the CNF Certification Suite app is running.
	CnfCertSuitePodName *string `json:"cnfCertSuitePodName,omitempty"`
	// Report holds the results and information related to the CNF Certification Suite run.
	Report *CertsuiteReport `json:"report,omitempty"`
}

type CnfPod struct {
	Name       string   `json:"name,omitempty"`
	Namespace  string   `json:"namespace,omitempty"`
	Containers []string `json:"containers,omitempty"`
}

type CnfResource struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

type CnfTargets struct {
	Namespaces        []string      `json:"namespaces,omitempty"`
	Nodes             []string      `json:"nodes,omitempty"`
	Pods              []CnfPod      `json:"pods,omitempty"`
	Deployments       []CnfResource `json:"deployments,omitempty"`
	StatefulSets      []CnfResource `json:"statefulSets,omitempty"`
	Csvs              []CnfResource `json:"csvs,omitempty"`
	Crds              []string      `json:"crds,omitempty"`
	Services          []CnfResource `json:"services,omitempty"`
	HelmChartReleases []CnfResource `json:"helmChartReleases,omitempty"`
}

// CertsuiteReport defines the desired state of CertsuiteReport
type CertsuiteReport struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	//+kubebuilder:validation:Enum=pass;skip;fail;error
	Verdict             string                       `json:"verdict"`
	OcpVersion          string                       `json:"ocpVersion"`
	CnfCertSuiteVersion string                       `json:"cnfCertSuiteVersion"`
	CnfTargets          CnfTargets                   `json:"cnfTargets,omitempty"`
	Summary             CertsuiteReportStatusSummary `json:"summary"`
	Results             []TestCaseResult             `json:"results"`
}

type TargetResource map[string]string

type TargetResources struct {
	Compliant    []TargetResource `json:"compliant,omitempty"`
	NonCompliant []TargetResource `json:"nonCompliant,omitempty"`
}

// TestCaseResult holds a test case result
type TestCaseResult struct {
	TestCaseName string `json:"testCaseName"`
	Description  string `json:"description"`
	//+kubebuilder:validation:Enum=passed;skipped;failed;error
	Result          string           `json:"result"`
	Reason          string           `json:"reason,omitempty"`
	Remediation     string           `json:"remediation,omitempty"`
	Logs            string           `json:"logs,omitempty"`
	TargetResources *TargetResources `json:"targetResources,omitempty"`
}

type CertsuiteReportStatusSummary struct {
	Total   int `json:"total"`
	Passed  int `json:"passed"`
	Skipped int `json:"skipped"`
	Failed  int `json:"failed"`
	Errored int `json:"errored"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="CertsuiteRun current status"
//+kubebuilder:printcolumn:name="Verdict",type="string",JSONPath=".status.report.verdict"

// CertsuiteRun is the Schema for the certsuiteruns API
type CertsuiteRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CertsuiteRunSpec   `json:"spec,omitempty"`
	Status CertsuiteRunStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CertsuiteRunList contains a list of CertsuiteRun
type CertsuiteRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CertsuiteRun `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CertsuiteRun{}, &CertsuiteRunList{})
}
