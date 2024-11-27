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

package controller

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	// appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/util/retry"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	cnfcertificationsv1alpha1 "github.com/redhat-best-practices-for-k8s/certsuite-operator/api/v1alpha1"
	cnfcertjob "github.com/redhat-best-practices-for-k8s/certsuite-operator/internal/controller/cnf-cert-job"
	"github.com/redhat-best-practices-for-k8s/certsuite-operator/internal/controller/definitions"
	controllerlogger "github.com/redhat-best-practices-for-k8s/certsuite-operator/internal/controller/logger"
)

var sideCarImage string

// CertsuiteRunReconciler reconciles a CertsuiteRun object
type CertsuiteRunReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

var (
	// certificationRuns maps a certificationRun to a pod name
	certificationRuns map[types.NamespacedName]string = map[types.NamespacedName]string{}
	// Holds an autoincremental CNF Cert Suite pod id
	certSuitePodID int
	// sets controller's logger.
	logger = controllerlogger.New()
)

const (
	checkInterval              = 5 * time.Second
	defaultCnfCertSuiteTimeout = time.Hour
)

// +kubebuilder:rbac:groups=best-practices-for-k8s.openshift.io,namespace=certsuite-operator,resources=certsuiteruns,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=best-practices-for-k8s.openshift.io,namespace=certsuite-operator,resources=certsuiteruns/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=best-practices-for-k8s.openshift.io,namespace=certsuite-operator,resources=certsuiteruns/finalizers,verbs=update

// +kubebuilder:rbac:groups="",namespace=certsuite-operator,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",namespace=certsuite-operator,resources=secrets;configMaps,verbs=get;list;watch
// +kubebuilder:rbac:groups="",namespace=certsuite-operator,resources=namespaces;services;configMaps,verbs=create;delete

// +kubebuilder:rbac:groups="console.openshift.io",resources=consoleplugins,verbs=create; delete
// +kubebuilder:rbac:groups="apps",namespace=certsuite-operator,resources=deployments,verbs=create;get;list;watch;delete

func ignoreUpdatePredicate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(_ event.UpdateEvent) bool {
			// Ignore updates to CR
			return false
		},
	}
}

// Helper method that updates the status of the CertsuiteRun CR. It uses
// the reconciler's client to Get an updated object first using the namespacedName fields.
// Then it calls the statusSetterFn that should update the required fields and finally
// calls de client's Update function to upload the updated object to the cluster.
func (r *CertsuiteRunReconciler) updateStatus(
	namespacedName types.NamespacedName,
	statusSetterFn func(currStatus *cnfcertificationsv1alpha1.CertsuiteRunStatus),
) error {
	runCR := cnfcertificationsv1alpha1.CertsuiteRun{}
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err := r.Get(context.TODO(), namespacedName, &runCR)
		if err != nil {
			return err
		}

		// Call the generic status updater func to set the new values.
		statusSetterFn(&runCR.Status)

		err = r.Status().Update(context.Background(), &runCR)
		if err != nil {
			return err
		}

		return nil
	})

	if retryErr != nil {
		logger.Errorf("Failed to update CertsuiteRun's %s status after retries: %v", namespacedName, retryErr)
		return retryErr
	}

	return nil
}

// // Updates CertsuiteRun.Status.Phase corresponding to a given status
func (r *CertsuiteRunReconciler) updateStatusPhase(namespacedName types.NamespacedName, phase cnfcertificationsv1alpha1.StatusPhase) error {
	return r.updateStatus(namespacedName, func(status *cnfcertificationsv1alpha1.CertsuiteRunStatus) {
		status.Phase = phase
	})
}

func getJobRunTimeThreshold(timeoutStr string) time.Duration {
	jobRunTimeThreshold, err := time.ParseDuration(timeoutStr)
	if err != nil {
		logger.Info("Couldn't extarct job run timeout, setting default timeout.")
		return defaultCnfCertSuiteTimeout
	}
	return jobRunTimeThreshold
}

func (r *CertsuiteRunReconciler) waitForCertSuitePodToComplete(certSuitePodNamespacedName types.NamespacedName, timeOut time.Duration) (exitStatusCode int32, err error) {
	for startTime := time.Now(); time.Since(startTime) < timeOut; {
		certSuitePod := corev1.Pod{}
		err = r.Get(context.TODO(), certSuitePodNamespacedName, &certSuitePod)
		if err != nil {
			return 0, err
		}

		switch certSuitePod.Status.Phase {
		case corev1.PodSucceeded:
			logger.Info("Cnf job pod has completed successfully.")
			return 0, nil
		case corev1.PodFailed:
			logger.Info("Cnf job pod has completed with failure.")
			exitStatus, err := getCertSuiteContainerExitStatus(&certSuitePod)
			if err != nil {
				return 0, err
			}
			return exitStatus, nil
		default:
			logger.Infof("Cnf job pod is running. Current status: %s", certSuitePod.Status.Phase)
			time.Sleep(checkInterval)
		}
	}

	return 0, fmt.Errorf("timeout (%s) reached while waiting for cert suite pod %v to finish", timeOut, certSuitePodNamespacedName)
}

func getCertSuiteContainerExitStatus(certSuitePod *corev1.Pod) (int32, error) {
	for i := range certSuitePod.Status.ContainerStatuses {
		containerStatus := &certSuitePod.Status.ContainerStatuses[i]
		if containerStatus.Name == definitions.CnfCertSuiteContainerName {
			return containerStatus.State.Terminated.ExitCode, nil
		}
	}

	return 0, fmt.Errorf("failed to get cert suite exit status: container not found in pod %s (ns %s)", certSuitePod.Name, certSuitePod.Namespace)
}

func (r *CertsuiteRunReconciler) handleEndOfCnfCertSuiteRun(runCrName, certSuitePodName, namespace, reqTimeout string) {
	certSuitePodNamespacedName := types.NamespacedName{Name: certSuitePodName, Namespace: namespace}
	runCrNamespacedName := types.NamespacedName{Name: runCrName, Namespace: namespace}

	certSuiteTimeout := getJobRunTimeThreshold(reqTimeout)
	certSuiteExitStatusCode, err := r.waitForCertSuitePodToComplete(certSuitePodNamespacedName, certSuiteTimeout)
	if err != nil {
		logger.Errorf("failed to handle end of cert suite run: %v", err)
	}

	// cnf-cert-job has terminated - checking exit status of cert suite
	if certSuiteExitStatusCode == 0 {
		logger.Info("CNF Cert job has finished running.")
		err = r.updateStatusPhase(runCrNamespacedName, definitions.CertsuiteRunStatusPhaseJobFinished)
	} else {
		logger.Info("CNF Cert job encountered an error. Exit status: ", certSuiteExitStatusCode)
		err = r.updateStatusPhase(runCrNamespacedName, definitions.CertsuiteRunStatusPhaseJobError)
	}

	if err != nil {
		logger.Errorf("Failed to update status field Phase of CR %s: %v", runCrNamespacedName, err)
	}
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CertsuiteRun object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
//
//nolint:funlen
func (r *CertsuiteRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger.Info("Reconciling CertsuiteRun CRD.")

	runCrNamespacedName := types.NamespacedName{Name: req.Name, Namespace: req.Namespace}
	var runCR cnfcertificationsv1alpha1.CertsuiteRun
	if getErr := r.Get(ctx, req.NamespacedName, &runCR); getErr != nil {
		logger.Infof("CertsuiteRun CR %s (ns %s) not found.", req.Name, req.NamespacedName)
		if podName, exist := certificationRuns[runCrNamespacedName]; exist {
			logger.Infof("CertsuiteRun has been deleted. Removing the associated CNF Cert job pod %v", podName)
			deleteErr := r.Delete(context.TODO(), &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: podName, Namespace: req.Namespace}})
			if deleteErr != nil {
				logger.Errorf("Failed to remove CNF Cert Job pod %s in namespace %s: %w", req.Name, req.Namespace, deleteErr)
			}
			delete(certificationRuns, runCrNamespacedName)
		}
		return ctrl.Result{}, client.IgnoreNotFound(getErr)
	}

	// Check if pod has already been created for given CersuiteRun CR.
	if runCR.Status.CnfCertSuitePodName != nil && *runCR.Status.CnfCertSuitePodName != "" {
		logger.Infof("Waiting for certsuite job pod %s to finish running", *runCR.Status.CnfCertSuitePodName)
		r.handleEndOfCnfCertSuiteRun(runCR.Name, *runCR.Status.CnfCertSuitePodName, runCR.Namespace, runCR.Spec.TimeOut)
		return ctrl.Result{}, nil
	}

	logger.Infof("New CNF Certification Job run requested: %v", runCrNamespacedName)

	certSuitePodID++
	certSuitePodName := fmt.Sprintf("%s-%d", definitions.CnfCertPodNamePrefix, certSuitePodID)

	// Store the new run & associated CNF Cert pod name
	certificationRuns[runCrNamespacedName] = certSuitePodName

	logger.Infof("Running CNF Certification Suite container (job id=%d) with labels %q, log level %q and timeout: %q",
		certSuitePodID, runCR.Spec.LabelsFilter, runCR.Spec.LogLevel, runCR.Spec.TimeOut)

	// Launch the pod with the CNF Cert Suite container plus the sidecar container to fetch the results.
	err := r.updateStatusPhase(runCrNamespacedName, cnfcertificationsv1alpha1.StatusPhaseCertSuiteDeploying)
	if err != nil {
		logger.Errorf("Failed to set status field Phase %s to CR %s: %v",
			cnfcertificationsv1alpha1.StatusPhaseCertSuiteDeploying, runCrNamespacedName, err)
		return ctrl.Result{}, nil
	}

	logger.Info("Creating CNF Cert job pod")
	cnfCertJobPod, err := cnfcertjob.New(
		cnfcertjob.WithPodName(certSuitePodName),
		cnfcertjob.WithNamespace(req.Namespace),
		cnfcertjob.WithCertSuiteConfigRunName(runCR.Name),
		cnfcertjob.WithLabelsFilter(runCR.Spec.LabelsFilter),
		cnfcertjob.WithLogLevel(runCR.Spec.LogLevel),
		cnfcertjob.WithTimeOut(runCR.Spec.TimeOut),
		cnfcertjob.WithConfigMap(runCR.Spec.ConfigMapName),
		cnfcertjob.WithPreflightSecret(runCR.Spec.PreflightSecretName),
		cnfcertjob.WithSideCarApp(sideCarImage),
		cnfcertjob.WithEnableDataCollection(strconv.FormatBool(runCR.Spec.EnableDataCollection)),
		cnfcertjob.WithOwnerReference(runCR.UID, runCR.Name, runCR.Kind, runCR.APIVersion),
	)
	if err != nil {
		logger.Errorf("Failed to create CNF Cert job pod spec: %w", err)
		if updateErr := r.updateStatusPhase(runCrNamespacedName, cnfcertificationsv1alpha1.StatusPhaseCertSuiteDeployError); updateErr != nil {
			logger.Errorf("Failed to set status field Phase %s to CR %s: %v", cnfcertificationsv1alpha1.StatusPhaseCertSuiteDeploying, runCrNamespacedName, updateErr)
		}
		return ctrl.Result{}, nil
	}

	err = r.Create(ctx, cnfCertJobPod)
	if err != nil {
		logger.Errorf("Failed to create CNF Cert job pod: %w", err)
		if updateErr := r.updateStatusPhase(runCrNamespacedName, cnfcertificationsv1alpha1.StatusPhaseCertSuiteDeployError); updateErr != nil {
			logger.Errorf("Failed to set status field Phase %s to CR %s: %v", cnfcertificationsv1alpha1.StatusPhaseCertSuiteDeployError, runCrNamespacedName, updateErr)
		}
		return ctrl.Result{}, nil
	}

	err = r.updateStatus(runCrNamespacedName, func(status *cnfcertificationsv1alpha1.CertsuiteRunStatus) {
		status.Phase = cnfcertificationsv1alpha1.StatusPhaseCertSuiteRunning
		status.CnfCertSuitePodName = &certSuitePodName
	})
	if err != nil {
		logger.Errorf("Failed to set status field Phase %s and podName %s to CR %s: %v",
			cnfcertificationsv1alpha1.StatusPhaseCertSuiteRunning, certSuitePodName, runCrNamespacedName, err)
		return ctrl.Result{}, nil
	}

	logger.Infof("Running CNF Cert job pod %s, triggered by CR %v", certSuitePodName, runCrNamespacedName)

	return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
}

func (r *CertsuiteRunReconciler) generateSinglePluginResourceObj(filePath, ns string, decoder runtime.Decoder) (client.Object, error) {
	yamlFile, err := os.ReadFile(filePath)
	if err != nil {
		logger.Errorf("failed to read plugin resource file: %s, err: %v", filePath, err)
		return nil, err
	}

	obj, _, err := decoder.Decode(yamlFile, nil, nil)
	if err != nil {
		logger.Errorf("failed to decode plugin resources yaml file, err: %v", err)
		return nil, err
	}

	clientObj := obj.(client.Object)
	clientObj.SetNamespace(ns)
	return clientObj, nil
}

func (r *CertsuiteRunReconciler) generatePluginResourcesObjs() ([]client.Object, error) {
	var pluginDir = "/plugin"

	// Read all  plugin's resources (written in yaml files)
	yamlFiles, err := os.ReadDir(pluginDir)
	if err != nil {
		logger.Errorf("failed to read plugin resources directory, err: %v", err)
		return nil, err
	}

	// Get controller's ns to set plugin in same ns
	controllerNS, found := os.LookupEnv(definitions.ControllerNamespaceEnvVar)
	if !found {
		return nil, fmt.Errorf("controller ns env var %q not found", definitions.ControllerNamespaceEnvVar)
	}

	// Iterate over all plugin's resources
	pluginObjList := []client.Object{}
	decoder := serializer.NewCodecFactory(r.Scheme).UniversalDeserializer()
	for _, file := range yamlFiles {
		yamlfilepath := filepath.Join(pluginDir, file.Name())
		obj, err := r.generateSinglePluginResourceObj(yamlfilepath, controllerNS, decoder)
		if err != nil {
			return nil, err
		}
		pluginObjList = append(pluginObjList, obj)
	}
	return pluginObjList, nil
}

func (r *CertsuiteRunReconciler) ApplyOperationOnPluginResources(op func(obj client.Object) error) error {
	// Generate plugin resources as objects
	pluginObjsList, err := r.generatePluginResourcesObjs()
	if err != nil {
		return fmt.Errorf("failed to generate plugin resources: %v", err)
	}

	// Apply given operation on plugin resources
	for _, obj := range pluginObjsList {
		err = op(obj)
		if err != nil {
			logger.Errorf("failed to apply operation on plugin resource, err: %v", err)
			return err
		}
	}

	return nil
}

func (r *CertsuiteRunReconciler) HandleConsolePlugin(done chan error) error {
	// Create console plugin resources
	err := r.ApplyOperationOnPluginResources(func(obj client.Object) error {
		return r.Create(context.Background(), obj)
	})
	if err != nil {
		return fmt.Errorf("failed to create plugin, err: %v", err)
	}
	logger.Info("Operator's console plugin was installed successfully.")

	// handle console plugin resources in operator termination
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigs
		done <- r.ApplyOperationOnPluginResources(func(obj client.Object) error {
			return r.Delete(context.Background(), obj)
		})
	}()

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CertsuiteRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	logger.Info("Setting up CertsuiteRunReconciler's manager.")

	var found bool
	sideCarImage, found = os.LookupEnv(definitions.SideCarImageEnvVar)
	if !found {
		return fmt.Errorf("sidecar app img env var %q not found", definitions.SideCarImageEnvVar)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&cnfcertificationsv1alpha1.CertsuiteRun{}).
		WithEventFilter(ignoreUpdatePredicate()).
		Complete(r)
}
