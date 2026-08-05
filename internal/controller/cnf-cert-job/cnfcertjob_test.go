package cnfcertjob

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/redhat-best-practices-for-k8s/certsuite-operator/internal/controller/definitions"
)

func TestNew(t *testing.T) {
	t.Run("no options produces valid base pod", func(t *testing.T) {
		pod, err := New()
		require.NoError(t, err)
		require.NotNil(t, pod)
		assert.Len(t, pod.Spec.Containers, 2)
		assert.Equal(t, definitions.CnfCertSuiteSidecarContainerName, pod.Spec.Containers[0].Name)
		assert.Equal(t, definitions.CnfCertSuiteContainerName, pod.Spec.Containers[1].Name)
		assert.Equal(t, corev1.RestartPolicy("Never"), pod.Spec.RestartPolicy)
		assert.Equal(t, clusterAccessServiceAccountName, pod.Spec.ServiceAccountName)
	})

	t.Run("options applied in order", func(t *testing.T) {
		pod, err := New(
			WithPodName("test-pod"),
			WithNamespace("test-ns"),
		)
		require.NoError(t, err)
		assert.Equal(t, "test-pod", pod.Name)
		assert.Equal(t, "test-ns", pod.Namespace)
	})

	t.Run("failing option returns error", func(t *testing.T) {
		failingOption := func(_ *corev1.Pod) error {
			return assert.AnError
		}
		pod, err := New(failingOption)
		assert.Error(t, err)
		assert.Nil(t, pod)
	})

	t.Run("first failing option short-circuits", func(t *testing.T) {
		called := false
		pod, err := New(
			func(_ *corev1.Pod) error { return assert.AnError },
			func(_ *corev1.Pod) error { called = true; return nil },
		)
		assert.Error(t, err)
		assert.Nil(t, pod)
		assert.False(t, called)
	})
}

func TestWithPodName(t *testing.T) {
	pod, err := New(WithPodName("my-pod"))
	require.NoError(t, err)
	assert.Equal(t, "my-pod", pod.Name)
}

func TestWithNamespace(t *testing.T) {
	pod, err := New(WithNamespace("my-ns"))
	require.NoError(t, err)
	assert.Equal(t, "my-ns", pod.Namespace)
}

func TestWithCertSuiteConfigRunName(t *testing.T) {
	t.Run("adds RUN_CR_NAME env var to sidecar", func(t *testing.T) {
		pod, err := New(WithCertSuiteConfigRunName("run-1"))
		require.NoError(t, err)
		sidecar := getSideCarAppContainer(pod)
		require.NotNil(t, sidecar)

		lastEnv := sidecar.Env[len(sidecar.Env)-1]
		assert.Equal(t, "run-1", lastEnv.Value)
	})

	t.Run("errors when sidecar container missing", func(t *testing.T) {
		pod := &corev1.Pod{}
		err := WithCertSuiteConfigRunName("run-1")(pod)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "side Car app Container is not found")
	})
}

func TestWithLabelsFilter(t *testing.T) {
	t.Run("appends labels filter args", func(t *testing.T) {
		pod, err := New(WithLabelsFilter("networking"))
		require.NoError(t, err)
		c := getCnfCertSuiteContainer(pod)
		require.NotNil(t, c)
		assert.Contains(t, c.Args, "-l")
		assert.Contains(t, c.Args, "networking")
	})

	t.Run("errors when certsuite container missing", func(t *testing.T) {
		pod := &corev1.Pod{}
		err := WithLabelsFilter("networking")(pod)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cnf cert suite Container is not found")
	})
}

func TestWithLogLevel(t *testing.T) {
	t.Run("appends log level args", func(t *testing.T) {
		pod, err := New(WithLogLevel("debug"))
		require.NoError(t, err)
		c := getCnfCertSuiteContainer(pod)
		require.NotNil(t, c)
		assert.Contains(t, c.Args, "--log-level")
		assert.Contains(t, c.Args, "debug")
	})

	t.Run("errors when certsuite container missing", func(t *testing.T) {
		pod := &corev1.Pod{}
		err := WithLogLevel("debug")(pod)
		assert.Error(t, err)
	})
}

func TestWithTimeOut(t *testing.T) {
	t.Run("appends timeout args", func(t *testing.T) {
		pod, err := New(WithTimeOut("2h"))
		require.NoError(t, err)
		c := getCnfCertSuiteContainer(pod)
		require.NotNil(t, c)
		assert.Contains(t, c.Args, "--timeout")
		assert.Contains(t, c.Args, "2h")
	})

	t.Run("errors when certsuite container missing", func(t *testing.T) {
		pod := &corev1.Pod{}
		err := WithTimeOut("2h")(pod)
		assert.Error(t, err)
	})
}

func TestWithConfigMap(t *testing.T) {
	pod, err := New(WithConfigMap("my-config"))
	require.NoError(t, err)

	var found bool
	for _, v := range pod.Spec.Volumes {
		if v.Name == volumeNameConfig && v.ConfigMap != nil {
			assert.Equal(t, "my-config", v.ConfigMap.Name)
			found = true
		}
	}
	assert.True(t, found, "config volume not found")
}

func TestWithPreflightSecret(t *testing.T) {
	t.Run("nil secret is no-op", func(t *testing.T) {
		pod, err := New(WithPreflightSecret(nil))
		require.NoError(t, err)
		for _, v := range pod.Spec.Volumes {
			assert.NotEqual(t, volumeNamePreflightDockerconf, v.Name)
		}
	})

	t.Run("non-nil secret adds volume and mount", func(t *testing.T) {
		secretName := "my-secret"
		pod, err := New(WithPreflightSecret(&secretName))
		require.NoError(t, err)

		var volumeFound bool
		for _, v := range pod.Spec.Volumes {
			if v.Name == volumeNamePreflightDockerconf && v.Secret != nil {
				assert.Equal(t, "my-secret", v.Secret.SecretName)
				volumeFound = true
			}
		}
		assert.True(t, volumeFound, "preflight secret volume not found")

		c := getCnfCertSuiteContainer(pod)
		require.NotNil(t, c)
		var mountFound bool
		for _, m := range c.VolumeMounts {
			if m.Name == volumeNamePreflightDockerconf {
				assert.Equal(t, definitions.CnfPreflightConfigFolder, m.MountPath)
				assert.True(t, m.ReadOnly)
				mountFound = true
			}
		}
		assert.True(t, mountFound, "preflight volume mount not found")
	})
}

func TestWithSideCarApp(t *testing.T) {
	t.Run("sets sidecar image", func(t *testing.T) {
		pod, err := New(WithSideCarApp("my-sidecar:v1"))
		require.NoError(t, err)
		sidecar := getSideCarAppContainer(pod)
		require.NotNil(t, sidecar)
		assert.Equal(t, "my-sidecar:v1", sidecar.Image)
	})

	t.Run("errors when sidecar container missing", func(t *testing.T) {
		pod := &corev1.Pod{}
		err := WithSideCarApp("img")(pod)
		assert.Error(t, err)
	})
}

func TestWithEnableDataCollection(t *testing.T) {
	t.Run("appends data collection arg", func(t *testing.T) {
		pod, err := New(WithEnableDataCollection("true"))
		require.NoError(t, err)
		c := getCnfCertSuiteContainer(pod)
		require.NotNil(t, c)
		assert.Contains(t, c.Args, "--enable-data-collection")
		assert.Contains(t, c.Args, "true")
	})

	t.Run("errors when certsuite container missing", func(t *testing.T) {
		pod := &corev1.Pod{}
		err := WithEnableDataCollection("true")(pod)
		assert.Error(t, err)
	})
}

func TestWithOwnerReference(t *testing.T) {
	pod, err := New(WithOwnerReference(types.UID("uid-123"), "owner", "CertsuiteRun", "v1alpha1"))
	require.NoError(t, err)
	require.Len(t, pod.OwnerReferences, 1)
	assert.Equal(t, "uid-123", string(pod.OwnerReferences[0].UID))
	assert.Equal(t, "owner", pod.OwnerReferences[0].Name)
	assert.Equal(t, "CertsuiteRun", pod.OwnerReferences[0].Kind)
	assert.Equal(t, "v1alpha1", pod.OwnerReferences[0].APIVersion)
}

func TestGetSideCarAppContainer(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		pod := newInitialJobPod()
		c := getSideCarAppContainer(pod)
		require.NotNil(t, c)
		assert.Equal(t, definitions.CnfCertSuiteSidecarContainerName, c.Name)
	})

	t.Run("not found", func(t *testing.T) {
		pod := &corev1.Pod{}
		c := getSideCarAppContainer(pod)
		assert.Nil(t, c)
	})
}

func TestGetCnfCertSuiteContainer(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		pod := newInitialJobPod()
		c := getCnfCertSuiteContainer(pod)
		require.NotNil(t, c)
		assert.Equal(t, definitions.CnfCertSuiteContainerName, c.Name)
	})

	t.Run("not found", func(t *testing.T) {
		pod := &corev1.Pod{}
		c := getCnfCertSuiteContainer(pod)
		assert.Nil(t, c)
	})
}

func TestNewInitialJobPod(t *testing.T) {
	pod := newInitialJobPod()

	assert.Len(t, pod.Spec.Containers, 2)
	assert.Equal(t, certsuiteContainerImage, pod.Spec.Containers[1].Image)
	assert.NotEmpty(t, pod.Spec.Containers[1].Command)
	assert.Contains(t, pod.Spec.Containers[1].Args, "run")
	assert.Contains(t, pod.Spec.Containers[1].Args, "--config-file")

	assert.Len(t, pod.Spec.Volumes, 1)
	assert.Equal(t, volumeNameOutput, pod.Spec.Volumes[0].Name)
	assert.NotNil(t, pod.Spec.Volumes[0].EmptyDir)
}

func TestFullOptionChain(t *testing.T) {
	secretName := "preflight-secret"
	pod, err := New(
		WithPodName("test-pod"),
		WithNamespace("test-ns"),
		WithCertSuiteConfigRunName("run-1"),
		WithLabelsFilter("networking"),
		WithLogLevel("debug"),
		WithTimeOut("2h"),
		WithConfigMap("my-config"),
		WithPreflightSecret(&secretName),
		WithSideCarApp("sidecar:latest"),
		WithEnableDataCollection("true"),
		WithOwnerReference("uid-1", "owner", "CertsuiteRun", "v1alpha1"),
	)
	require.NoError(t, err)
	assert.Equal(t, "test-pod", pod.Name)
	assert.Equal(t, "test-ns", pod.Namespace)
	assert.Len(t, pod.OwnerReferences, 1)

	sidecar := getSideCarAppContainer(pod)
	require.NotNil(t, sidecar)
	assert.Equal(t, "sidecar:latest", sidecar.Image)

	certsuite := getCnfCertSuiteContainer(pod)
	require.NotNil(t, certsuite)
	assert.Contains(t, certsuite.Args, "-l")
	assert.Contains(t, certsuite.Args, "--log-level")
	assert.Contains(t, certsuite.Args, "--timeout")
	assert.Contains(t, certsuite.Args, "--enable-data-collection")

	assert.Len(t, pod.Spec.Volumes, 3)
}
