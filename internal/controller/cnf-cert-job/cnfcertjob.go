package cnfcertjob

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/redhat-best-practices-for-k8s/certsuite-operator/internal/controller/definitions"
)

const (
	// Be careful when changing this SA name.
	// 1. It must match the flag --extra-service-accounts in "make bundle".
	// 2. The prefix is "certsuite-". It should match the field namePrefix field in config/default/kustomization.yaml.
	clusterAccessServiceAccountName = "certsuite-cluster-access"

	certsuiteContainerImage = "quay.io/redhat-best-practices-for-k8s/certsuite:unstable"
)

func New(options ...func(*corev1.Pod) error) (*corev1.Pod, error) {
	jobPod := newInitialJobPod()

	for _, o := range options {
		err := o(jobPod)
		if err != nil {
			return nil, err
		}
	}
	return jobPod, nil
}

//nolint:funlen
func newInitialJobPod() *corev1.Pod {
	return &corev1.Pod{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Spec: corev1.PodSpec{
			ServiceAccountName: clusterAccessServiceAccountName,
			RestartPolicy:      "Never",
			Containers: []corev1.Container{
				{
					Name: definitions.CnfCertSuiteSidecarContainerName,
					Env: []corev1.EnvVar{
						{
							Name: "MY_POD_NAME",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: "metadata.name",
								},
							},
						},
						{
							Name: "MY_POD_NAMESPACE",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: "metadata.namespace",
								},
							},
						},
						{
							Name:  definitions.SideCarResultsFolderEnvVar,
							Value: definitions.CnfCertSuiteResultsFolder,
						},
					},
					ImagePullPolicy: "IfNotPresent",
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "certsuite-output",
							ReadOnly:  true,
							MountPath: definitions.CnfCertSuiteResultsFolder,
						},
					},
				},
				{
					Name:    definitions.CnfCertSuiteContainerName,
					Image:   certsuiteContainerImage,
					Command: []string{"certsuite"},
					Args: []string{"run", "--output-dir", definitions.CnfCertSuiteResultsFolder,
						"--config-file", definitions.CnfCertSuiteConfigFilePath,
						"--preflight-dockerconfig", definitions.PreflightDockerConfigFilePath,
						"--intrusive", "false",
					},
					ImagePullPolicy: "Always",
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "certsuite-output",
							MountPath: definitions.CnfCertSuiteResultsFolder,
						},
						{
							Name:      "certsuite-config",
							ReadOnly:  true,
							MountPath: definitions.CnfCnfCertSuiteConfigFolder,
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "certsuite-output",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
		Status: corev1.PodStatus{},
	}
}

func WithPodName(podName string) func(*corev1.Pod) error {
	return func(p *corev1.Pod) error {
		p.Name = podName
		return nil
	}
}

func WithNamespace(namespace string) func(*corev1.Pod) error {
	return func(p *corev1.Pod) error {
		p.Namespace = namespace
		return nil
	}
}

func WithCertSuiteConfigRunName(certSuiteConfigRunName string) func(*corev1.Pod) error {
	return func(p *corev1.Pod) error {
		envVar := corev1.EnvVar{Name: "RUN_CR_NAME", Value: certSuiteConfigRunName}
		sideCarContainer := getSideCarAppContainer(p)
		if sideCarContainer == nil {
			return fmt.Errorf("side Car app Container is not found in pod %s", p.Name)
		}
		sideCarContainer.Env = append(sideCarContainer.Env, envVar)
		return nil
	}
}

func WithLabelsFilter(labelsFilter string) func(*corev1.Pod) error {
	return func(p *corev1.Pod) error {
		cnfCertSuiteContainer := getCnfCertSuiteContainer(p)
		if cnfCertSuiteContainer == nil {
			return fmt.Errorf("cnf cert suite Container is not found in pod %s", p.Name)
		}
		cnfCertSuiteContainer.Args = append(cnfCertSuiteContainer.Args, "-l", labelsFilter)
		return nil
	}
}

func WithLogLevel(loglevel string) func(*corev1.Pod) error {
	return func(p *corev1.Pod) error {
		cnfCertSuiteContainer := getCnfCertSuiteContainer(p)
		if cnfCertSuiteContainer == nil {
			return fmt.Errorf("cnf cert suite Container is not found in pod %s", p.Name)
		}
		cnfCertSuiteContainer.Args = append(cnfCertSuiteContainer.Args, "--log-level", loglevel)
		return nil
	}
}

func WithTimeOut(timeout string) func(*corev1.Pod) error {
	return func(p *corev1.Pod) error {
		cnfCertSuiteContainer := getCnfCertSuiteContainer(p)
		if cnfCertSuiteContainer == nil {
			return fmt.Errorf("cnf cert suite Container is not found in pod %s", p.Name)
		}
		cnfCertSuiteContainer.Args = append(cnfCertSuiteContainer.Args, "--timeout", timeout)
		return nil
	}
}

func WithConfigMap(configMapName string) func(*corev1.Pod) error {
	return func(p *corev1.Pod) error {
		Volume := corev1.Volume{
			Name: "certsuite-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: configMapName,
					},
				},
			},
		}
		p.Spec.Volumes = append(p.Spec.Volumes, Volume)
		return nil
	}
}

func WithPreflightSecret(preflightSecretName *string) func(*corev1.Pod) error {
	return func(p *corev1.Pod) error {
		if preflightSecretName == nil {
			return nil
		}

		Volume := corev1.Volume{
			Name: "certsuite-preflight-dockerconfig",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: *preflightSecretName,
				},
			},
		}
		p.Spec.Volumes = append(p.Spec.Volumes, Volume)

		cnfCertCuiteContainer := getCnfCertSuiteContainer(p)
		volumeMount := corev1.VolumeMount{
			Name:      "certsuite-preflight-dockerconfig",
			ReadOnly:  true,
			MountPath: definitions.CnfPreflightConfigFolder,
		}
		cnfCertCuiteContainer.VolumeMounts = append(cnfCertCuiteContainer.VolumeMounts, volumeMount)
		return nil
	}
}

func WithSideCarApp(sideCarAppImage string) func(*corev1.Pod) error {
	return func(p *corev1.Pod) error {
		sideCarContainer := getSideCarAppContainer(p)
		if sideCarContainer == nil {
			return fmt.Errorf("side Car app Container is not found in pod %s", p.Name)
		}
		sideCarContainer.Image = sideCarAppImage
		return nil
	}
}

func WithEnableDataCollection(enableDataCollection string) func(*corev1.Pod) error {
	return func(p *corev1.Pod) error {
		cnfCertSuiteContainer := getCnfCertSuiteContainer(p)
		if cnfCertSuiteContainer == nil {
			return fmt.Errorf("cnf cert suite Container is not found in pod %s", p.Name)
		}
		cnfCertSuiteContainer.Args = append(cnfCertSuiteContainer.Args, "--enable-data-collection", enableDataCollection)
		return nil
	}
}

func WithOwnerReference(ownerUID types.UID, ownerName, ownerKind, ownerAPIVersion string) func(*corev1.Pod) error {
	return func(p *corev1.Pod) error {
		ownerReference := &metav1.OwnerReference{
			APIVersion: ownerAPIVersion,
			Kind:       ownerKind,
			Name:       ownerName,
			UID:        ownerUID,
		}
		p.OwnerReferences = []metav1.OwnerReference{*ownerReference}
		return nil
	}
}

func getSideCarAppContainer(p *corev1.Pod) *corev1.Container {
	for i := range p.Spec.Containers {
		if p.Spec.Containers[i].Name == definitions.CnfCertSuiteSidecarContainerName {
			return &p.Spec.Containers[i]
		}
	}
	return nil
}

func getCnfCertSuiteContainer(p *corev1.Pod) *corev1.Container {
	for i := range p.Spec.Containers {
		if p.Spec.Containers[i].Name == definitions.CnfCertSuiteContainerName {
			return &p.Spec.Containers[i]
		}
	}
	return nil
}
