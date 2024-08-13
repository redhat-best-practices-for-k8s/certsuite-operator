package main

import (
	"context"
	"encoding/json"
	"os"
	"time"

	cnfcertificationsv1alpha1 "github.com/redhat-best-practices-for-k8s/certsuite-operator/api/v1alpha1"
	"github.com/redhat-best-practices-for-k8s/certsuite-operator/certsuite-sidecar/app/claim"
	cnfcertsuitereport "github.com/redhat-best-practices-for-k8s/certsuite-operator/certsuite-sidecar/app/cnf-cert-suite-report"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	podNamespaceEnvVar = "MY_POD_NAMESPACE"
	runCrNameEnvVar    = "RUN_CR_NAME"
)

const (
	sideCarResultsFolderEnvVar = "TNF_RESULTS_FOLDER"
	claimFileName              = "claim.json"
	multiplier                 = 5
)

func handleClaimFile(k8sClient client.Client) {
	namespace := os.Getenv(podNamespaceEnvVar)
	runCRname := os.Getenv(runCrNameEnvVar)

	claimFolder := os.Getenv(sideCarResultsFolderEnvVar)
	claimFilePath := claimFolder + "/" + claimFileName

	logrus.Infof("Claim file: %v", claimFilePath)
	logrus.Infof("CnfCertificationSuiteRun CR: %s/%s", namespace, runCRname)

	for {
		_, err := os.Stat(claimFilePath)
		if os.IsNotExist(err) {
			logrus.Warnf("Claim file not found yet. Waiting 5 secs...")
			time.Sleep(multiplier * time.Second)
			continue
		}

		// Wait extra 5 secs to give time to the writer to finish.
		// TODO: Use file locking mechanism between writer/reader of this file.
		time.Sleep(multiplier * time.Second)

		logrus.Infof("Claim file found at %v", claimFilePath)
		claimBytes, err := os.ReadFile(claimFilePath)
		if err != nil {
			logrus.Fatalf("Failed to read claim file %s: %v", claimFilePath, err)
		}

		claimContent := claim.Schema{}
		err = json.Unmarshal(claimBytes, &claimContent)
		if err != nil {
			logrus.Fatalf("Failed to unmarshal claim json: %v", err)
		}

		// Get the CnfCertificationSuiteRun CR
		runCR := cnfcertificationsv1alpha1.CnfCertificationSuiteRun{}
		err = k8sClient.Get(context.TODO(),
			types.NamespacedName{
				Name:      runCRname,
				Namespace: namespace},
			&runCR)

		if err != nil {
			logrus.Fatalf("Failed to get CnfCertificationSuiteRun CR %s (ns %s)", runCRname, namespace)
		}

		cnfcertsuitereport.SetRunCRStatus(&runCR, &claimContent)

		err = k8sClient.Status().Update(context.TODO(), &runCR)
		if err != nil {
			logrus.Fatalf("Failed to update CnfCertificationSuiteRun.Status object object: %v", err)
		}

		logrus.Infof("CnfCertificationSuiteRun CR's status updated successfully with results:\n%v", runCR.Status.Report.Results)
		break
	}
}

// This CNF Certification sidecar container expects to be running in the same
// pod as the CNF Cert Suite container.
//
// The sidecar will wait for TNF_CNF_CERT_TIMEOUT secs until the claim.json output is present in a
// shared volume folder. Then it will parse it to get the list of test cases
// results, and creates the CR CnfCertificationSuiteReport
func main() {
	scheme := runtime.NewScheme()
	err := clientgoscheme.AddToScheme(scheme)
	if err != nil {
		logrus.Fatalf("failed to add to scheme: %v", err)
	}

	err = cnfcertificationsv1alpha1.AddToScheme(scheme)
	if err != nil {
		logrus.Fatalf("failed to add cnfcertificationsv1alpha1 to scheme: %v", err)
	}

	kubeConfigFile := os.Getenv("HOME") + "/.kube/config"
	_, err = os.Stat(kubeConfigFile)
	if err != nil {
		kubeConfigFile = ""
	}

	k8sRestConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigFile)
	if err != nil {
		logrus.Fatalf("Failed to get incluster rest config: %v", err)
	}

	DefaultTimeout := 10 * time.Second
	k8sRestConfig.Timeout = DefaultTimeout

	k8sClient, err := client.New(k8sRestConfig, client.Options{Scheme: scheme})
	if err != nil {
		logrus.Fatalf("Failed to get k8s client: %v", err)
	}

	handleClaimFile(k8sClient)
}
