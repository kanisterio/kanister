package param

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/Masterminds/sprig"
	. "gopkg.in/check.v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	crfake "github.com/kanisterio/kanister/pkg/client/clientset/versioned/fake"
	"github.com/kanisterio/kanister/pkg/kube"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type ParamsSuite struct {
	cli       kubernetes.Interface
	namespace string
	pvc       string
}

var _ = Suite(&ParamsSuite{})

func (s *ParamsSuite) SetUpSuite(c *C) {
	cli, err := kube.NewClient()
	c.Assert(err, IsNil)
	s.cli = cli
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kanisterparamstest-",
		},
	}
	cns, err := s.cli.Core().Namespaces().Create(ns)
	c.Assert(err, IsNil)
	s.namespace = cns.Name
}

func (s *ParamsSuite) SetUpTest(c *C) {
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kanisterparamtest-",
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: *resource.NewQuantity(1, resource.BinarySI),
				},
			},
		},
	}
	cPVC, err := s.cli.CoreV1().PersistentVolumeClaims(s.namespace).Create(pvc)
	c.Assert(err, IsNil)
	s.pvc = cPVC.Name
}

func (s *ParamsSuite) TearDownSuite(c *C) {
	if s.namespace != "" {
		s.cli.Core().Namespaces().Delete(s.namespace, nil)
	}
}

func (s *ParamsSuite) TearDownTest(c *C) {
	err := s.cli.CoreV1().PersistentVolumeClaims(s.namespace).Delete(s.pvc, &metav1.DeleteOptions{})
	c.Assert(err, IsNil)
}

const ssSpec = `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: %s
spec:
  replicas: 1
  serviceName: fake-svc
  selector:
    matchLabels:
      app: fake-app
  template:
    metadata:
      labels:
        app: fake-app
    spec:
      containers:
        - name: test-container
          image: alpine:3.6
          command: ["tail"]
          args: ["-f", "/dev/null"]
          volumeMounts:
            - name: test-vol
              mountPath: /mnt/data/%s
      volumes:
        - name: test-vol
          persistentVolumeClaim:
            claimName: %s
`

func (s *ParamsSuite) TestFetchStatefulSetParams(c *C) {
	ctx := context.Background()
	name := strings.ToLower(c.TestName())
	name = strings.Replace(name, ".", "", 1)
	spec := fmt.Sprintf(ssSpec, name, name, s.pvc)
	ss, err := kube.CreateStatefulSet(ctx, s.cli, s.namespace, spec)
	c.Assert(err, IsNil)
	err = kube.WaitOnStatefulSetReady(ctx, s.cli, ss.Namespace, ss.Name)
	c.Assert(err, IsNil)

	ssp, err := fetchStatefulSetParams(ctx, s.cli, s.namespace, name)
	c.Assert(err, IsNil)
	c.Assert(ssp, DeepEquals, &StatefulSetParams{
		Name:       name,
		Namespace:  s.namespace,
		Pods:       []string{name + "-0"},
		Containers: [][]string{{"test-container"}},
		PersistentVolumeClaims: map[string]map[string]string{
			name + "-0": map[string]string{
				s.pvc + "-" + name + "-0": "/mnt/data/" + name,
			},
		},
	})

}

const deploySpec = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
spec:
  replicas: 1
  selector:
    matchLabels:
      app: fake-app
  template:
    metadata:
      labels:
        app: fake-app
    spec:
      containers:
        - name: test-container
          image: alpine:3.6
          command: ["tail"]
          args: ["-f", "/dev/null"]
          volumeMounts:
            - name: test-vol
              mountPath: /mnt/data/%s
      volumes:
        - name: test-vol
          persistentVolumeClaim:
            claimName: %s
`

func (s *ParamsSuite) TestFetchDeploymentParams(c *C) {
	ctx := context.Background()
	name := strings.ToLower(c.TestName())
	name = strings.Replace(name, ".", "", 1)
	spec := fmt.Sprintf(deploySpec, name, name, s.pvc)
	d, err := kube.CreateDeployment(ctx, s.cli, s.namespace, spec)
	c.Assert(err, IsNil)
	err = kube.WaitOnDeploymentReady(ctx, s.cli, d.Namespace, d.Name)
	c.Assert(err, IsNil)

	dp, err := fetchDeploymentParams(ctx, s.cli, s.namespace, name)
	c.Assert(err, IsNil)
	c.Assert(dp.Namespace, Equals, s.namespace)
	c.Assert(dp.Pods, HasLen, 1)
	c.Assert(dp.Containers, DeepEquals, [][]string{{"test-container"}})
	c.Assert(dp.PersistentVolumeClaims, DeepEquals, map[string]map[string]string{
		dp.Pods[0]: map[string]string{
			s.pvc: "/mnt/data/" + name,
		},
	})
}

func (s *ParamsSuite) TestFetchPVCParams(c *C) {
	ctx := context.Background()
	testCases := []struct {
		name       string
		pvc        string
		errChecker Checker
	}{
		{"Valid", s.pvc, IsNil},
		{"Invalid", "foo-pvc", NotNil},
	}
	for _, tc := range testCases {
		_, err := fetchPVCParams(ctx, s.cli, s.namespace, tc.pvc)
		c.Check(err, tc.errChecker, Commentf("Test %s Failed!", tc.name))
	}
}

const cmSpec = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: %s-cm
data:
  someKey: some-value
`

func (s *ParamsSuite) TestNewTemplateParamsDeployment(c *C) {
	ctx := context.Background()
	name := strings.ToLower(c.TestName())
	name = strings.Replace(name, ".", "", 1)
	spec := fmt.Sprintf(deploySpec, name, name, s.pvc)
	d, err := kube.CreateDeployment(ctx, s.cli, s.namespace, spec)
	c.Assert(err, IsNil)
	err = kube.WaitOnDeploymentReady(ctx, s.cli, d.Namespace, d.Name)
	c.Assert(err, IsNil)

	s.testNewTemplateParams(ctx, c, crv1alpha1.ObjectReference{Name: name, Namespace: s.namespace, Kind: DeploymentKind})
}

func (s *ParamsSuite) TestNewTemplateParamsStatefulSet(c *C) {
	ctx := context.Background()
	name := strings.ToLower(c.TestName())
	name = strings.Replace(name, ".", "", 1)
	spec := fmt.Sprintf(ssSpec, name, name, s.pvc)
	ss, err := kube.CreateStatefulSet(ctx, s.cli, s.namespace, spec)
	c.Assert(err, IsNil)
	err = kube.WaitOnStatefulSetReady(ctx, s.cli, ss.Namespace, ss.Name)
	c.Assert(err, IsNil)

	s.testNewTemplateParams(ctx, c, crv1alpha1.ObjectReference{Name: name, Namespace: s.namespace, Kind: StatefulSetKind})
}

func (s *ParamsSuite) TestNewTemplateParamsPVC(c *C) {
	ctx := context.Background()
	s.testNewTemplateParams(ctx, c, crv1alpha1.ObjectReference{Name: s.pvc, Namespace: s.namespace, Kind: PVCKind})
}

func (s *ParamsSuite) TestNewTemplateParamsNamespace(c *C) {
	ctx := context.Background()
	s.testNewTemplateParams(ctx, c, crv1alpha1.ObjectReference{Name: s.namespace, Namespace: s.namespace, Kind: NamespaceKind})
}

func (s *ParamsSuite) TestNewTemplateParamsUnstructured(c *C) {
	ctx := context.Background()
	// Lookup the "default" serviceaccount in the test namespace
	s.testNewTemplateParams(ctx, c, crv1alpha1.ObjectReference{Name: "default", Namespace: s.namespace, Group: "", APIVersion: "v1", Resource: "serviceaccounts"})
}

func (s *ParamsSuite) testNewTemplateParams(ctx context.Context, c *C, object crv1alpha1.ObjectReference) {
	spec := fmt.Sprintf(cmSpec, object.Name)
	cm, err := kube.CreateConfigMap(ctx, s.cli, s.namespace, spec)
	c.Assert(err, IsNil)
	c.Assert(cm, NotNil)

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret-name",
			Namespace: s.namespace,
			Labels:    map[string]string{"app": "fake-app"},
		},
		Data: map[string][]byte{
			"key":   []byte("myKey"),
			"value": []byte("myValue"),
		},
	}
	prof := &crv1alpha1.Profile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "profName",
			Namespace: s.namespace,
		},
		Credential: crv1alpha1.Credential{
			Type: crv1alpha1.CredentialTypeKeyPair,
			KeyPair: &crv1alpha1.KeyPair{
				IDField:     "key",
				SecretField: "value",
				Secret: crv1alpha1.ObjectReference{
					Name:      "secret-name",
					Namespace: s.namespace,
				},
			},
		},
	}
	_, err = s.cli.CoreV1().Secrets(s.namespace).Create(secret)
	c.Assert(err, IsNil)
	defer s.cli.CoreV1().Secrets(s.namespace).Delete("secret-name", &metav1.DeleteOptions{})

	_, err = s.cli.CoreV1().Secrets(s.namespace).Get("secret-name", metav1.GetOptions{})
	c.Assert(err, IsNil)

	crCli := crfake.NewSimpleClientset()
	_, err = crCli.CrV1alpha1().Profiles(s.namespace).Create(prof)
	c.Assert(err, IsNil)
	_, err = crCli.CrV1alpha1().Profiles(s.namespace).Get("profName", metav1.GetOptions{})
	c.Assert(err, IsNil)

	as := crv1alpha1.ActionSpec{
		Object: object,
		ConfigMaps: map[string]crv1alpha1.ObjectReference{
			"myCM": crv1alpha1.ObjectReference{
				Name:      object.Name + "-cm",
				Namespace: s.namespace,
			},
		},
		Profile: &crv1alpha1.ObjectReference{
			Name:      "profName",
			Namespace: s.namespace,
		},
		Options: map[string]string{
			"podName": "some-pod",
		},
	}
	var template string
	switch object.Kind {
	case DeploymentKind:
		template = "{{ .Deployment.Name }}"
	case StatefulSetKind:
		template = "{{ .StatefulSet.Name }}"
	case PVCKind:
		template = "{{ .PVC.Name }}"
	case NamespaceKind:
		template = "{{ .Namespace.Name }}"
	default:
		template = "{{ .Object.metadata.name }}"
	}

	artsTpl := map[string]crv1alpha1.Artifact{
		"my-art": crv1alpha1.Artifact{KeyValue: map[string]string{
			"my-key": "{{ .ConfigMaps.myCM.Data.someKey }}"},
		},
		"my-time": crv1alpha1.Artifact{KeyValue: map[string]string{
			"my-time": "{{ .Time }}"},
		},
		"kindArtifact": crv1alpha1.Artifact{KeyValue: map[string]string{"my-key": template}},
	}
	artsTpl["kindArtifact"] = crv1alpha1.Artifact{KeyValue: map[string]string{"my-key": template}}

	tp, err := New(ctx, s.cli, crCli, as)
	c.Assert(err, IsNil)
	c.Assert(tp.ConfigMaps["myCM"].Data, DeepEquals, map[string]string{"someKey": "some-value"})
	c.Assert(tp.Options, DeepEquals, map[string]string{"podName": "some-pod"})

	arts, err := RenderArtifacts(artsTpl, *tp)
	c.Assert(err, IsNil)
	c.Assert(arts["my-art"], DeepEquals, crv1alpha1.Artifact{KeyValue: map[string]string{"my-key": "some-value"}})
	_, err = time.Parse(timeFormat, arts["my-time"].KeyValue["my-time"])
	c.Assert(err, IsNil)
	c.Assert(arts["kindArtifact"], DeepEquals, crv1alpha1.Artifact{KeyValue: map[string]string{"my-key": object.Name}})
}

func (s *ParamsSuite) TestfetchKVSecretCredential(c *C) {
	ctx := context.Background()
	for _, tc := range []struct {
		secret  *v1.Secret
		kvs     *crv1alpha1.KeyPair
		checker Checker
		cred    *Credential
	}{
		{
			secret:  &v1.Secret{},
			kvs:     &crv1alpha1.KeyPair{},
			cred:    nil,
			checker: NotNil,
		},
		{
			secret: &v1.Secret{
				Data: map[string][]byte{
					"myKey":   []byte("foo"),
					"myValue": []byte("bar"),
				},
			},
			kvs: &crv1alpha1.KeyPair{
				IDField:     "myKey",
				SecretField: "myValue",
				Secret:      crv1alpha1.ObjectReference{},
			},
			cred: &Credential{
				Type: CredentialTypeKeyPair,
				KeyPair: &KeyPair{
					ID:     "foo",
					Secret: "bar",
				},
			},
			checker: IsNil,
		},
	} {
		cli := fake.NewSimpleClientset(tc.secret)
		cred, err := fetchKeyPairCredential(ctx, cli, tc.kvs)
		c.Assert(err, tc.checker)
		c.Assert(cred, DeepEquals, tc.cred)
	}
}

func (s *ParamsSuite) TestProfile(c *C) {
	ss := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ssName",
			Namespace: s.namespace,
			Labels:    map[string]string{"app": "fake-app"},
		},
	}
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "podName",
			Namespace: s.namespace,
			Labels:    map[string]string{"app": "fake-app"},
		},
	}
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secretName",
			Namespace: s.namespace,
			Labels:    map[string]string{"app": "fake-app"},
		},
		Data: map[string][]byte{
			"key":   []byte("myKey"),
			"value": []byte("myValue"),
		},
	}
	cli := fake.NewSimpleClientset(ss, pod, secret)
	_, err := cli.AppsV1().StatefulSets("").Get("", metav1.GetOptions{})
	c.Assert(err, IsNil)
	_, err = cli.CoreV1().Pods("").Get("", metav1.GetOptions{})
	c.Assert(err, IsNil)
	_, err = cli.CoreV1().Secrets("").Get("", metav1.GetOptions{})
	c.Assert(err, IsNil)

	prof := &crv1alpha1.Profile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "profName",
			Namespace: s.namespace,
		},
		Credential: crv1alpha1.Credential{
			Type: crv1alpha1.CredentialTypeKeyPair,
			KeyPair: &crv1alpha1.KeyPair{
				IDField:     "key",
				SecretField: "value",
				Secret: crv1alpha1.ObjectReference{
					Name:      "secretName",
					Namespace: s.namespace,
				},
			},
		},
	}
	as := &crv1alpha1.ActionSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "asName",
			Namespace: s.namespace,
		},
		Spec: &crv1alpha1.ActionSetSpec{
			Actions: []crv1alpha1.ActionSpec{
				crv1alpha1.ActionSpec{
					Object: crv1alpha1.ObjectReference{
						Kind:      "StatefulSet",
						Name:      "ssName",
						Namespace: s.namespace,
					},
					Profile: &crv1alpha1.ObjectReference{},
				},
			},
		},
	}
	crCli := crfake.NewSimpleClientset()
	_, err = crCli.CrV1alpha1().ActionSets(s.namespace).Create(as)
	c.Assert(err, IsNil)
	_, err = crCli.CrV1alpha1().ActionSets(s.namespace).Get("", metav1.GetOptions{})
	c.Assert(err, IsNil)
	_, err = crCli.CrV1alpha1().Profiles(s.namespace).Create(prof)
	c.Assert(err, IsNil)
	_, err = crCli.CrV1alpha1().Profiles(s.namespace).Get("", metav1.GetOptions{})
	c.Assert(err, IsNil)

	ctx := context.Background()
	tp, err := New(ctx, cli, crCli, as.Spec.Actions[0])
	c.Assert(err, IsNil)
	c.Assert(tp.Profile, NotNil)
	c.Assert(tp.Profile, DeepEquals, &Profile{
		Location: crv1alpha1.Location{},
		Credential: Credential{
			Type: CredentialTypeKeyPair,
			KeyPair: &KeyPair{
				ID:     "myKey",
				Secret: "myValue",
			},
		},
	})
}

func (s *ParamsSuite) TestPhaseParams(c *C) {
	ctx := context.Background()
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret-name",
			Namespace: s.namespace,
			Labels:    map[string]string{"app": "fake-app"},
		},
		Data: map[string][]byte{
			"key":   []byte("myKey"),
			"value": []byte("myValue"),
		},
	}
	prof := &crv1alpha1.Profile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "profName",
			Namespace: s.namespace,
		},
		Credential: crv1alpha1.Credential{
			Type: crv1alpha1.CredentialTypeKeyPair,
			KeyPair: &crv1alpha1.KeyPair{
				IDField:     "key",
				SecretField: "value",
				Secret: crv1alpha1.ObjectReference{
					Name:      "secret-name",
					Namespace: s.namespace,
				},
			},
		},
	}
	_, err := s.cli.CoreV1().Secrets(s.namespace).Create(secret)
	c.Assert(err, IsNil)
	defer s.cli.CoreV1().Secrets(s.namespace).Delete("secret-name", &metav1.DeleteOptions{})

	_, err = s.cli.CoreV1().Secrets(s.namespace).Get("secret-name", metav1.GetOptions{})
	c.Assert(err, IsNil)

	crCli := crfake.NewSimpleClientset()
	_, err = crCli.CrV1alpha1().Profiles(s.namespace).Create(prof)
	c.Assert(err, IsNil)
	_, err = crCli.CrV1alpha1().Profiles(s.namespace).Get("profName", metav1.GetOptions{})
	c.Assert(err, IsNil)
	as := crv1alpha1.ActionSpec{
		Object: crv1alpha1.ObjectReference{
			Name:      s.pvc,
			Namespace: s.namespace,
			Kind:      PVCKind,
		},
		Profile: &crv1alpha1.ObjectReference{
			Name:      "profName",
			Namespace: s.namespace,
		},
	}
	tp, err := New(ctx, s.cli, crCli, as)
	c.Assert(err, IsNil)
	c.Assert(tp.Phases, IsNil)
	err = InitPhaseParams(ctx, s.cli, tp, "backup", nil)
	c.Assert(err, IsNil)
	UpdatePhaseParams(ctx, tp, "backup", map[string]interface{}{"version": "0.14.0"})
	c.Assert(tp.Phases, HasLen, 1)
	c.Assert(tp.Phases["backup"], NotNil)
}

func (s *ParamsSuite) TestRenderingPhaseParams(c *C) {
	ctx := context.Background()
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret-dfss",
			Namespace: "ns1",
		},
		StringData: map[string]string{
			"myKey":   "foo",
			"myValue": "bar",
		},
	}
	cli := fake.NewSimpleClientset(secret)
	secretRef := map[string]crv1alpha1.ObjectReference{
		"authSecret": crv1alpha1.ObjectReference{
			Kind:      SecretKind,
			Name:      secret.Name,
			Namespace: secret.Namespace,
		},
	}
	tp := TemplateParams{}
	err := InitPhaseParams(ctx, cli, &tp, "backup", secretRef)
	c.Assert(err, IsNil)
	UpdatePhaseParams(ctx, &tp, "backup", map[string]interface{}{"replicas": 2})
	for _, tc := range []struct {
		arg      string
		expected string
	}{
		{
			"{{ .Phases.backup.Output.replicas }}",
			"2",
		},
		{
			"{{ .Phases.backup.Secrets.authSecret.Namespace }}",
			"ns1",
		},
		{
			"{{ .Phases.backup.Secrets.authSecret.StringData.myValue }}",
			"bar",
		},
	} {
		t, err := template.New("config").Option("missingkey=error").Funcs(sprig.TxtFuncMap()).Parse(tc.arg)
		c.Assert(err, IsNil)
		buf := bytes.NewBuffer(nil)
		err = t.Execute(buf, tp)
		c.Assert(err, IsNil)
		c.Assert(buf.String(), Equals, tc.expected)
	}
}
