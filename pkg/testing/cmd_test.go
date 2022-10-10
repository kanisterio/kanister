package testing

import (
	"context"
	"fmt"
	"time"

	"gopkg.in/check.v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/kopia/command"
	"github.com/kanisterio/kanister/pkg/kopia/repository"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/testutil"
)

const (
	testPodName = "test-kopia-cmd-"
)

type KopiaCmdSuite struct {
	cli       kubernetes.Interface
	namespace string
}

var _ = check.Suite(&KopiaCmdSuite{})

func (s *KopiaCmdSuite) SetUpSuite(c *check.C) {
	config, err := kube.LoadConfig()
	c.Assert(err, check.IsNil)
	cli, err := kubernetes.NewForConfig(config)
	c.Assert(err, check.IsNil)
	s.cli = cli

	ctx := context.Background()
	ns := testutil.NewTestNamespace()
	ns.GenerateName = "kanister-datatest-"

	cns, err := s.cli.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)
	s.namespace = cns.GetName()

}

func (s *KopiaCmdSuite) TearDownSuite(c *check.C) {
	ctx := context.Background()
	if s.namespace != "" {
		_ = s.cli.CoreV1().Namespaces().Delete(ctx, s.namespace, metav1.DeleteOptions{})
	}
}

func (s *KopiaCmdSuite) TestRepositoryCreate(c *check.C) {
	locSecret, credSecret := s.createLocationAndCredSecrets(c)
	pod := s.startKanisterToolsPod(c, locSecret, credSecret)
	err := repository.CreateKopiaRepository(s.cli, s.namespace, pod.Name, pod.Spec.Containers[0].Name, command.RepositoryCommandArgs{
		CommandArgs: &command.CommandArgs{
			RepoPassword:   "test-pass123",
			ConfigFilePath: "/tmp/config",
			LogDirectory:   "/tmp/logs",
		},
		Hostname:        "test-hostname",
		Username:        "test-username",
		RepoPathPrefix:  fmt.Sprintf("test-path/test-%v", time.Now().Unix()),
		ContentCacheMB:  0,
		MetadataCacheMB: 0,
		CacheDirectory:  "/tmp/cache",
	})
	c.Assert(err, check.IsNil)
}

func (s *KopiaCmdSuite) startKanisterToolsPod(c *check.C, locSecret, credSecret *v1.Secret) *v1.Pod {
	volMounts := []v1.VolumeMount{
		{
			Name:      repository.LocationSecretVolumeMountName,
			MountPath: repository.LocationSecretMountPath,
		},
	}
	vols := []v1.Volume{
		{
			Name: repository.LocationSecretVolumeMountName,
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: locSecret.Name,
				},
			},
		},
	}
	if credSecret != nil {
		volMounts = append(volMounts, v1.VolumeMount{
			Name:      repository.CredsSecretVolumeMountName,
			MountPath: repository.CredsSecretMountPath,
		})
		vols = append(vols, v1.Volume{
			Name: repository.CredsSecretVolumeMountName,
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: credSecret.Name,
				},
			},
		})
	}
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: testPodName,
			Namespace:    s.namespace,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:         "kanister-tools",
					Image:        "ghcr.io/kanisterio/kanister-tools:0.82.0",
					Command:      []string{"sh", "-c", "tail -f /dev/null"},
					VolumeMounts: volMounts,
				},
			},
			Volumes: vols,
		},
	}
	pod, err := s.cli.CoreV1().Pods(s.namespace).Create(context.Background(), pod, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)
	err = kube.WaitForPodReady(context.Background(), s.cli, pod.Namespace, pod.Name)
	c.Assert(err, check.IsNil)
	return pod
}

func (s *KopiaCmdSuite) createLocationAndCredSecrets(c *check.C) (*v1.Secret, *v1.Secret) {
	ls := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "location-secret-",
		},
		StringData: map[string]string{
			"prefix": "test-prefix",
			"type":   "filestore",
		},
	}
	locSecret, err := s.cli.CoreV1().Secrets(s.namespace).Create(context.Background(), ls, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)

	return locSecret, nil
}
