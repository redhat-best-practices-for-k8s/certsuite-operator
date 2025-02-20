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
	"path/filepath"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	bestpracticesfork8sv1alpha1 "github.com/redhat-best-practices-for-k8s/certsuite-operator/api/v1alpha1"
	"github.com/redhat-best-practices-for-k8s/certsuite-operator/internal/controller/definitions"
	controllerlogger "github.com/redhat-best-practices-for-k8s/certsuite-operator/internal/controller/logger"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CertsuiteConsolePluginReconciler reconciles a CertsuiteConsolePlugin object
type CertsuiteConsolePluginReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

var pluginLogger = controllerlogger.New()

const (
	pluginResourcesCreationConditionType   = "PluginResourcesCreatedSuccessfully"
	pluginCreationSucceededConditionReason = "PluginCreationHasSucceeded"
	pluginCreationSucceededConditionMsg    = "Plugin Creation Has Succeeded."
)

func ignoreUpdateInCertsuiteConsolePlugin() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(_ event.UpdateEvent) bool {
			// Ignore updates to CR
			return false
		},
	}
}

func (r *CertsuiteConsolePluginReconciler) generateSinglePluginResourceObj(filePath, ns string, decoder runtime.Decoder) (client.Object, error) {
	yamlFile, err := os.ReadFile(filePath)
	if err != nil {
		pluginLogger.Errorf("failed to read plugin resource file: %s, err: %v", filePath, err)
		return nil, err
	}

	obj, _, err := decoder.Decode(yamlFile, nil, nil)
	if err != nil {
		pluginLogger.Errorf("failed to decode plugin resources yaml file, err: %v", err)
		return nil, err
	}

	clientObj := obj.(client.Object)
	clientObj.SetNamespace(ns)
	return clientObj, nil
}

func (r *CertsuiteConsolePluginReconciler) generatePluginResourcesObjs() ([]client.Object, error) {
	var pluginDir = "/plugin"

	// Read all  plugin's resources (written in yaml files)
	yamlFiles, err := os.ReadDir(pluginDir)
	if err != nil {
		pluginLogger.Errorf("failed to read plugin resources directory, err: %v", err)
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

func (r *CertsuiteConsolePluginReconciler) createConsolePluginResources(consolePluginCR *bestpracticesfork8sv1alpha1.CertsuiteConsolePlugin) error {
	pluginObjsList, err := r.generatePluginResourcesObjs()
	if err != nil {
		return fmt.Errorf("failed to generate plugin resources: %v", err)
	}

	condition := metav1.Condition{
		Type:    pluginResourcesCreationConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  pluginCreationSucceededConditionReason,
		Message: pluginCreationSucceededConditionMsg,
	}

	for _, obj := range pluginObjsList {
		objName := obj.GetName()
		objNamespace := obj.GetNamespace()
		objKind := obj.GetObjectKind().GroupVersionKind().Kind

		pluginLogger.Info("Creating console plugin resource", "namespace", objName, "name", objNamespace, "kind", objKind)

		if err = r.Create(context.Background(), obj); err != nil {
			if k8serrors.IsAlreadyExists(err) {
				pluginLogger.Info("Openshift Console plugin resource already exists", "namespace", objNamespace, "name", objName, "kind", objKind)
			} else {
				condition.Status = metav1.ConditionFalse
				condition.Reason = fmt.Sprintf("%sResourceCreationFailed", objKind)
				condition.Message = fmt.Sprintf("failed to create plugin resource %s of kind %s (ns %s): %v", objName, objKind, objNamespace, err)
			}
			break
		}
	}

	// Update status condition
	condition.LastTransitionTime = metav1.Now()
	consolePluginCR.Status.Conditions = append(consolePluginCR.Status.Conditions, condition)
	if err := r.Status().Update(context.TODO(), consolePluginCR); err != nil {
		return fmt.Errorf("failed to update console plugin CR's status conditions: %v", err)
	}

	if condition.Reason != pluginCreationSucceededConditionReason {
		return fmt.Errorf("failed to create console plugin: %s", condition.Message)
	}

	pluginLogger.Info("Operator's console plugin was installed successfully.")

	return nil
}

func (r *CertsuiteConsolePluginReconciler) deleteConsolePluginResources() error {
	pluginObjsList, err := r.generatePluginResourcesObjs()
	if err != nil {
		return fmt.Errorf("failed to generate plugin resources: %v", err)
	}

	for _, obj := range pluginObjsList {
		if err := r.Delete(context.Background(), obj); err != nil {
			return fmt.Errorf("error has occurred when trying to delete %s console plugin's resource: %v", obj.GetObjectKind(), err)
		}
	}
	return nil
}

//+kubebuilder:rbac:groups=best-practices-for-k8s.openshift.io,resources=certsuiteconsoleplugins,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=best-practices-for-k8s.openshift.io,resources=certsuiteconsoleplugins/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=best-practices-for-k8s.openshift.io,resources=certsuiteconsoleplugins/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CertsuiteConsolePlugin object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.3/pkg/reconcile
func (r *CertsuiteConsolePluginReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	pluginLogger.Info("Reconciling CertsuiteRun CRD.")

	// Get req CertsuiteConsolePlugin CR
	var consolePluginCR bestpracticesfork8sv1alpha1.CertsuiteConsolePlugin
	if getErr := r.Get(ctx, req.NamespacedName, &consolePluginCR); getErr != nil {
		pluginLogger.Infof("CertsuiteConsolePlugin CR %s has been deleted. Removing the console plugin resources.", req.NamespacedName)
		if err := r.deleteConsolePluginResources(); err != nil {
			return ctrl.Result{}, fmt.Errorf("unable to delete console plugin resources: %v", err)
		}

		return ctrl.Result{}, client.IgnoreNotFound(getErr)
	}

	// CertsuiteConsolePlugin CR has been created and it's the only CertsuiteConsolePlugin CR, create console plugin resources
	if err := r.createConsolePluginResources(&consolePluginCR); err != nil {
		return ctrl.Result{}, fmt.Errorf("error has occurred while handling console plugin: %v", err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CertsuiteConsolePluginReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&bestpracticesfork8sv1alpha1.CertsuiteConsolePlugin{}).
		WithEventFilter(ignoreUpdateInCertsuiteConsolePlugin()).
		Complete(r)
}
