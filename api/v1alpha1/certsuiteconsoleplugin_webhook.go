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
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const consolePluginCRsNumLimit = 1

// log is for logging in this package.
var certsuiteconsolepluginlog = logf.Log.WithName("certsuiteconsoleplugin-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *CertsuiteConsolePlugin) SetupWebhookWithManager(mgr ctrl.Manager) error {
	c = mgr.GetClient()

	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithValidator(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
//nolint:lll
//+kubebuilder:webhook:path=/validate-best-practices-for-k8s-openshift-io-v1alpha1-certsuiteconsoleplugin,mutating=false,failurePolicy=fail,sideEffects=None,groups=best-practices-for-k8s.openshift.io,resources=certsuiteconsoleplugins,verbs=create;update,versions=v1alpha1,name=vcertsuiteconsoleplugin.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = &CertsuiteConsolePlugin{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *CertsuiteConsolePlugin) ValidateCreate(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	certsuiteconsolepluginlog.Info("validate create", "name", r.Name)

	var consolePluginCRList CertsuiteConsolePluginList
	if err := c.List(context.TODO(), &consolePluginCRList, &client.ListOptions{}); err != nil {
		return nil, fmt.Errorf("unable to list CertsuiteConsolePlugin CRs (ns: %s): %v", r.Namespace, err)
	}

	// If there is more already at least one CertsuiteConsolePlugin CR return an error
	if len(consolePluginCRList.Items) >= consolePluginCRsNumLimit {
		return nil, fmt.Errorf("unable to create CR. certsuiteConsolePlugin CR count limit: %d, certsuiteConsolePlugin CR current count: %d", consolePluginCRsNumLimit, len(consolePluginCRList.Items))
	}

	// If no CertsuiteConsolePlugin CR exist, allow creation of new CR
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *CertsuiteConsolePlugin) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	_ = oldObj

	r = newObj.(*CertsuiteConsolePlugin)
	certsuiteconsolepluginlog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *CertsuiteConsolePlugin) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	r = obj.(*CertsuiteConsolePlugin)
	certsuiteconsolepluginlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
