package controller

import (
	"context"
	"fmt"
	"testing"
	"time"

	cnfcertificationsv1alpha1 "github.com/redhat-best-practices-for-k8s/certsuite-operator/api/v1alpha1"
	"github.com/redhat-best-practices-for-k8s/certsuite-operator/internal/controller/definitions"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// Creates a reconciler with a fake client to mock API calls.
func mockReconciler(objs []runtime.Object) *CertsuiteRunReconciler {
	s := scheme.Scheme
	s.AddKnownTypes(cnfcertificationsv1alpha1.GroupVersion, &cnfcertificationsv1alpha1.CertsuiteRun{})

	runCR := &cnfcertificationsv1alpha1.CertsuiteRun{}
	cl := fake.NewClientBuilder().WithRuntimeObjects(objs...).WithStatusSubresource(runCR).Build()

	return &CertsuiteRunReconciler{Client: cl, Scheme: s}
}

func Test_getJobRunTimeThreshold(t *testing.T) {
	tests := []struct {
		name       string
		timeoutStr string
		want       time.Duration
	}{
		{ // Test case #1 - Pass with given timeout
			name:       "Set timeout",
			timeoutStr: "2h",
			want:       2 * time.Hour,
		},
		{ // Test case #2 - Pass with default timeout as timeoutStr is an empty string
			name:       "Empty timeout",
			timeoutStr: "",
			want:       time.Hour,
		},
	}

	for _, tc := range tests {
		assert.Equal(t, getJobRunTimeThreshold(tc.timeoutStr), tc.want)
	}
}

func TestCertsuiteRunReconciler_updateStatus(t *testing.T) {
	tests := []struct {
		name                string
		statusSetterFn      func(currStatus *cnfcertificationsv1alpha1.CertsuiteRunStatus)
		statusCheckerFn     func(currStatus *cnfcertificationsv1alpha1.CertsuiteRunStatus) error
		runCRNamespacedName types.NamespacedName
		wantErr             bool
	}{
		{ // Test case #1 - Pass with exit status 0
			name: "Pass when updating phase",
			statusSetterFn: func(currStatus *cnfcertificationsv1alpha1.CertsuiteRunStatus) {
				currStatus.Phase = definitions.CertsuiteRunStatusPhaseJobFinished
			},
			statusCheckerFn: func(currStatus *cnfcertificationsv1alpha1.CertsuiteRunStatus) error {
				if currStatus.Phase != definitions.CertsuiteRunStatusPhaseJobFinished {
					return fmt.Errorf("CertsuiteRun status updated has failed. current status: %v, wanted status: %v",
						currStatus.Phase, definitions.CertsuiteRunStatusPhaseJobFinished)
				}
				return nil
			},
			runCRNamespacedName: types.NamespacedName{
				Name:      "cnf-run-sample",
				Namespace: "certsuite-operator",
			},
			wantErr: false,
		},
		{ // Test case #1 - Fail, error = certsuiteruns.best-practices-for-k8s.openshift.io "" not found
			name: "Fail updating phase",
			statusSetterFn: func(currStatus *cnfcertificationsv1alpha1.CertsuiteRunStatus) {
				currStatus.Phase = definitions.CertsuiteRunStatusPhaseJobFinished
			},
			statusCheckerFn: func(currStatus *cnfcertificationsv1alpha1.CertsuiteRunStatus) error {
				if currStatus.Phase != definitions.CertsuiteRunStatusPhaseJobFinished {
					return fmt.Errorf("CertsuiteRun status updated has failed. current status: %v, wanted status: %v",
						currStatus.Phase, definitions.CertsuiteRunStatusPhaseJobFinished)
				}
				return nil
			},
			runCRNamespacedName: types.NamespacedName{},
			wantErr:             true,
		},
	}

	runCR := &cnfcertificationsv1alpha1.CertsuiteRun{
		ObjectMeta: v1.ObjectMeta{
			Name:      "cnf-run-sample",
			Namespace: "certsuite-operator",
		},
		Status: cnfcertificationsv1alpha1.CertsuiteRunStatus{
			Phase: definitions.CertsuiteRunStatusCertSuiteRunning,
		},
	}
	for _, tc := range tests {
		r := mockReconciler([]runtime.Object{runCR})

		// check whether an error has occurred if expected, and hasn't occurred if not expected
		err := r.updateStatus(tc.runCRNamespacedName, tc.statusSetterFn)
		if (err != nil) != tc.wantErr {
			t.Errorf("CertsuiteRunReconciler.updateStatus() error = %v, wantErr %v", err, tc.wantErr)
		}

		// check if status was updated (if an error hasn't occurred)
		if err == nil {
			updatedRunCR := cnfcertificationsv1alpha1.CertsuiteRun{}
			err := r.Get(context.TODO(), tc.runCRNamespacedName, &updatedRunCR)
			if err != nil {
				t.Errorf("Error getting updated Run CR ")
			}
			err = tc.statusCheckerFn(&updatedRunCR.Status)
			if err != nil {
				t.Errorf("%s", err.Error())
			}
		}
	}
}
