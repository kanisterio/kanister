// Copyright 2019 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package param

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"text/template"
	"time"

	osapps "github.com/openshift/api/apps/v1"
	osversioned "github.com/openshift/client-go/apps/clientset/versioned"
	osfake "github.com/openshift/client-go/apps/clientset/versioned/fake"
	. "gopkg.in/check.v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	fakedyncli "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	crfake "github.com/kanisterio/kanister/pkg/client/clientset/versioned/fake"
	"github.com/kanisterio/kanister/pkg/ksprig"
	"github.com/kanisterio/kanister/pkg/kube"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type ParamsSuite struct {
	cli       kubernetes.Interface
	dynCli    dynamic.Interface
	namespace string
	pvc       string
	osCli     osversioned.Interface
}

var _ = Suite(&ParamsSuite{})

func (s *ParamsSuite) SetUpSuite(c *C) {
	cli, err := kube.NewClient()
	c.Assert(err, IsNil)
	s.cli = cli
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kanisterparamstest-",
		},
	}
	cns, err := s.cli.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	s.namespace = cns.Name
	s.dynCli = fakedyncli.NewSimpleDynamicClient(scheme.Scheme, cns)
}

func (s *ParamsSuite) SetUpTest(c *C) {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kanisterparamtest-",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceName(corev1.ResourceStorage): resource.MustParse("1Gi"),
				},
			},
		},
	}
	cPVC, err := s.cli.CoreV1().PersistentVolumeClaims(s.namespace).Create(context.TODO(), pvc, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	s.pvc = cPVC.Name
}

func (s *ParamsSuite) TearDownSuite(c *C) {
	if s.namespace != "" {
		_ = s.cli.CoreV1().Namespaces().Delete(context.TODO(), s.namespace, metav1.DeleteOptions{})
	}
}

func (s *ParamsSuite) TearDownTest(c *C) {
	err := s.cli.CoreV1().PersistentVolumeClaims(s.namespace).Delete(context.TODO(), s.pvc, metav1.DeleteOptions{})
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
			name + "-0": {
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
		dp.Pods[0]: {
			s.pvc: "/mnt/data/" + name,
		},
	})
}

func (s *ParamsSuite) TestFetchDeploymentConfigParams(c *C) {
	ok, err := kube.IsOSAppsGroupAvailable(context.Background(), s.cli.Discovery())
	c.Assert(err, IsNil)
	if !ok {
		c.Skip("Skipping test since this only runs on OpenShift")
	}

	cfg, err := kube.LoadConfig()
	c.Assert(err, IsNil)

	s.osCli, err = osversioned.NewForConfig(cfg)
	c.Assert(err, IsNil)

	depConf := newDeploymentConfig()
	c.Assert(err, IsNil)

	// create a deploymentconfig
	ctx := context.Background()
	dc, err := s.osCli.AppsV1().DeploymentConfigs(s.namespace).Create(ctx, depConf, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	// wait for deploymentconfig to be ready
	err = kube.WaitOnDeploymentConfigReady(ctx, s.osCli, s.cli, dc.Namespace, dc.Name)
	c.Assert(err, IsNil)

	// get again achieve optimistic concurrency
	newDep, err := s.osCli.AppsV1().DeploymentConfigs(s.namespace).Get(ctx, dc.Name, metav1.GetOptions{})
	c.Assert(err, IsNil)

	// edit the deploymentconfig
	newDep.Spec.Template.Spec.Containers[0].Name = "newname"
	// update the deploymentconfig
	updatedDC, err := s.osCli.AppsV1().DeploymentConfigs(s.namespace).Update(ctx, newDep, metav1.UpdateOptions{})
	c.Assert(err, IsNil)

	// once updated, it will take some time to new replicationcontroller and pods to be up and running
	// wait for deploymentconfig to be reay again
	err = kube.WaitOnDeploymentConfigReady(ctx, s.osCli, s.cli, dc.Namespace, updatedDC.Name)
	c.Assert(err, IsNil)

	// fetch the deploymentconfig params
	dconf, err := fetchDeploymentConfigParams(ctx, s.cli, s.osCli, s.namespace, updatedDC.Name)

	c.Assert(err, IsNil)
	c.Assert(dconf.Namespace, Equals, s.namespace)
	c.Assert(dconf.Pods, HasLen, 1)
	c.Assert(dconf.Containers, DeepEquals, [][]string{{"newname"}})

	// let's scale the deployment config and try things
	dConfig, err := s.osCli.AppsV1().DeploymentConfigs(s.namespace).Get(ctx, dc.Name, metav1.GetOptions{})
	c.Assert(err, IsNil)
	// scale the replicas to 3
	dConfig.Spec.Replicas = 3
	updated, err := s.osCli.AppsV1().DeploymentConfigs(s.namespace).Update(ctx, dConfig, metav1.UpdateOptions{})
	c.Assert(err, IsNil)
	// wait for deploymentconfig to be ready
	err = kube.WaitOnDeploymentConfigReady(ctx, s.osCli, s.cli, s.namespace, updated.Name)
	c.Assert(err, IsNil)

	// fetch the deploymentconfig params
	dconfParams, err := fetchDeploymentConfigParams(ctx, s.cli, s.osCli, s.namespace, updated.Name)
	c.Assert(err, IsNil)
	c.Assert(dconfParams.Namespace, Equals, s.namespace)
	// number of pods should be chnanged to 3
	c.Assert(dconfParams.Pods, HasLen, 3)
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

	s.testNewTemplateParams(ctx, c, s.getDynamicClient(c, d), crv1alpha1.ObjectReference{Name: name, Namespace: s.namespace, Kind: DeploymentKind})
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

	s.testNewTemplateParams(ctx, c, s.getDynamicClient(c, ss), crv1alpha1.ObjectReference{Name: name, Namespace: s.namespace, Kind: StatefulSetKind})
}

func (s *ParamsSuite) TestNewTemplateParamsPVC(c *C) {
	ctx := context.Background()
	pvc, err := s.cli.CoreV1().PersistentVolumeClaims(s.namespace).Get(context.TODO(), s.pvc, metav1.GetOptions{})
	c.Assert(err, IsNil)
	s.testNewTemplateParams(ctx, c, s.getDynamicClient(c, pvc), crv1alpha1.ObjectReference{Name: s.pvc, Namespace: s.namespace, Kind: PVCKind})
}

func (s *ParamsSuite) TestNewTemplateParamsNamespace(c *C) {
	ctx := context.Background()
	s.testNewTemplateParams(ctx, c, s.getDynamicClient(c), crv1alpha1.ObjectReference{Name: s.namespace, Namespace: s.namespace, Kind: NamespaceKind})
}

func (s *ParamsSuite) TestNewTemplateParamsUnstructured(c *C) {
	ctx := context.Background()
	// Lookup the "default" serviceaccount in the test namespace
	sa, err := s.cli.CoreV1().ServiceAccounts(s.namespace).Get(context.TODO(), "default", metav1.GetOptions{})
	c.Assert(err, IsNil)
	s.testNewTemplateParams(ctx, c, s.getDynamicClient(c, sa), crv1alpha1.ObjectReference{Name: "default", Namespace: s.namespace, Group: "", APIVersion: "v1", Resource: "serviceaccounts"})
}

func (s *ParamsSuite) getDynamicClient(c *C, objects ...runtime.Object) dynamic.Interface {
	ns, err := s.cli.CoreV1().Namespaces().Get(context.TODO(), s.namespace, metav1.GetOptions{})
	c.Assert(err, IsNil)
	objects = append(objects, ns)
	return fakedyncli.NewSimpleDynamicClient(scheme.Scheme, objects...)
}

func (s *ParamsSuite) testNewTemplateParams(ctx context.Context, c *C, dynCli dynamic.Interface, object crv1alpha1.ObjectReference) {
	spec := fmt.Sprintf(cmSpec, object.Name)
	cm, err := kube.CreateConfigMap(ctx, s.cli, s.namespace, spec)
	c.Assert(err, IsNil)
	c.Assert(cm, NotNil)

	secret := &corev1.Secret{
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
	_, err = s.cli.CoreV1().Secrets(s.namespace).Create(ctx, secret, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	defer func() {
		_ = s.cli.CoreV1().Secrets(s.namespace).Delete(context.TODO(), "secret-name", metav1.DeleteOptions{})
	}()

	_, err = s.cli.CoreV1().Secrets(s.namespace).Get(ctx, "secret-name", metav1.GetOptions{})
	c.Assert(err, IsNil)

	osCli := osfake.NewSimpleClientset()

	crCli := crfake.NewSimpleClientset()
	_, err = crCli.CrV1alpha1().Profiles(s.namespace).Create(ctx, prof, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	_, err = crCli.CrV1alpha1().Profiles(s.namespace).Get(ctx, "profName", metav1.GetOptions{})
	c.Assert(err, IsNil)

	as := crv1alpha1.ActionSpec{
		Object: object,
		ConfigMaps: map[string]crv1alpha1.ObjectReference{
			"myCM": {
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
	unstructuredTemplate := "{{ .Object.metadata.name }}"
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
		"my-art": {KeyValue: map[string]string{
			"my-key": "{{ .ConfigMaps.myCM.Data.someKey }}"},
		},
		"my-time": {KeyValue: map[string]string{
			"my-time": "{{ .Time }}"},
		},
		"kindArtifact": {KeyValue: map[string]string{"my-key": template}},
	}
	artsTpl["kindArtifact"] = crv1alpha1.Artifact{KeyValue: map[string]string{"my-key": template}}
	artsTpl["objectNameArtifact"] = crv1alpha1.Artifact{KeyValue: map[string]string{"my-key": unstructuredTemplate}}

	tp, err := New(ctx, s.cli, dynCli, crCli, osCli, as)
	c.Assert(err, IsNil)
	c.Assert(tp.ConfigMaps["myCM"].Data, DeepEquals, map[string]string{"someKey": "some-value"})
	c.Assert(tp.Options, DeepEquals, map[string]string{"podName": "some-pod"})

	arts, err := RenderArtifacts(artsTpl, *tp)
	c.Assert(err, IsNil)
	c.Assert(arts["my-art"], DeepEquals, crv1alpha1.Artifact{KeyValue: map[string]string{"my-key": "some-value"}})
	_, err = time.Parse(timeFormat, arts["my-time"].KeyValue["my-time"])
	c.Assert(err, IsNil)
	c.Assert(arts["kindArtifact"], DeepEquals, crv1alpha1.Artifact{KeyValue: map[string]string{"my-key": object.Name}})
	c.Assert(arts["objectNameArtifact"], DeepEquals, crv1alpha1.Artifact{KeyValue: map[string]string{"my-key": object.Name}})
}

func (s *ParamsSuite) TestfetchKVSecretCredential(c *C) {
	ctx := context.Background()
	for _, tc := range []struct {
		secret  *corev1.Secret
		kvs     *crv1alpha1.KeyPair
		checker Checker
		cred    *Credential
	}{
		{
			secret:  &corev1.Secret{},
			kvs:     &crv1alpha1.KeyPair{},
			cred:    nil,
			checker: NotNil,
		},
		{
			secret: &corev1.Secret{
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
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "podName",
			Namespace: s.namespace,
			Labels:    map[string]string{"app": "fake-app"},
		},
	}
	secret := &corev1.Secret{
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
	ctx := context.Background()
	cli := fake.NewSimpleClientset(ss, pod, secret)
	dynCli := fakedyncli.NewSimpleDynamicClient(scheme.Scheme, ss)
	_, err := cli.AppsV1().StatefulSets("").List(ctx, metav1.ListOptions{})
	c.Assert(err, IsNil)
	_, err = cli.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	c.Assert(err, IsNil)
	_, err = cli.CoreV1().Secrets("").List(ctx, metav1.ListOptions{})
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
				{
					Object: crv1alpha1.ObjectReference{
						Kind:      "StatefulSet",
						Name:      "ssName",
						Namespace: s.namespace,
					},
					Profile: &crv1alpha1.ObjectReference{
						Name:      "profName",
						Namespace: s.namespace,
					},
				},
			},
		},
	}
	crCli := crfake.NewSimpleClientset()
	_, err = crCli.CrV1alpha1().ActionSets(s.namespace).Create(ctx, as, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	_, err = crCli.CrV1alpha1().ActionSets(s.namespace).List(ctx, metav1.ListOptions{})
	c.Assert(err, IsNil)
	_, err = crCli.CrV1alpha1().Profiles(s.namespace).Create(ctx, prof, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	_, err = crCli.CrV1alpha1().Profiles(s.namespace).List(ctx, metav1.ListOptions{})
	c.Assert(err, IsNil)

	osCli := osfake.NewSimpleClientset()

	tp, err := New(ctx, cli, dynCli, crCli, osCli, as.Spec.Actions[0])
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

func (s *ParamsSuite) TestParamsWithoutProfile(c *C) {
	ctx := context.Background()
	secret := &corev1.Secret{
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
	secret, err := s.cli.CoreV1().Secrets(s.namespace).Create(ctx, secret, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	defer func() {
		_ = s.cli.CoreV1().Secrets(s.namespace).Delete(context.TODO(), "secret-name", metav1.DeleteOptions{})
	}()

	_, err = s.cli.CoreV1().Secrets(s.namespace).Get(ctx, "secret-name", metav1.GetOptions{})
	c.Assert(err, IsNil)

	pvc, err := s.cli.CoreV1().PersistentVolumeClaims(s.namespace).Get(ctx, s.pvc, metav1.GetOptions{})
	c.Assert(err, IsNil)
	dynCli := s.getDynamicClient(c, pvc)
	crCli := crfake.NewSimpleClientset()
	osCli := osfake.NewSimpleClientset()
	as := crv1alpha1.ActionSpec{
		Object: crv1alpha1.ObjectReference{
			Name:      s.pvc,
			Namespace: s.namespace,
			Kind:      PVCKind,
		},
		Secrets: map[string]crv1alpha1.ObjectReference{
			"actionSetSecret": {
				Name:      secret.Name,
				Namespace: secret.Namespace,
			},
		},
	}
	tp, err := New(ctx, s.cli, dynCli, crCli, osCli, as)
	c.Assert(err, IsNil)
	c.Assert(tp, NotNil)
}

func (s *ParamsSuite) TestPhaseParams(c *C) {
	ctx := context.Background()
	secret := &corev1.Secret{
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
	secret, err := s.cli.CoreV1().Secrets(s.namespace).Create(ctx, secret, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	defer func() {
		_ = s.cli.CoreV1().Secrets(s.namespace).Delete(context.TODO(), "secret-name", metav1.DeleteOptions{})
	}()

	_, err = s.cli.CoreV1().Secrets(s.namespace).Get(ctx, "secret-name", metav1.GetOptions{})
	c.Assert(err, IsNil)

	pvc, err := s.cli.CoreV1().PersistentVolumeClaims(s.namespace).Get(ctx, s.pvc, metav1.GetOptions{})
	c.Assert(err, IsNil)
	dynCli := s.getDynamicClient(c, pvc)
	crCli := crfake.NewSimpleClientset()
	osCli := osfake.NewSimpleClientset()
	_, err = crCli.CrV1alpha1().Profiles(s.namespace).Create(ctx, prof, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	_, err = crCli.CrV1alpha1().Profiles(s.namespace).Get(ctx, "profName", metav1.GetOptions{})
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
		Secrets: map[string]crv1alpha1.ObjectReference{
			"actionSetSecret": {
				Name:      secret.Name,
				Namespace: secret.Namespace,
			},
		},
	}
	tp, err := New(ctx, s.cli, dynCli, crCli, osCli, as)
	c.Assert(err, IsNil)
	c.Assert(tp.Phases, IsNil)
	err = InitPhaseParams(ctx, s.cli, tp, "backup", nil)
	c.Assert(err, IsNil)
	UpdatePhaseParams(ctx, tp, "backup", map[string]interface{}{"version": "0.75.0"})
	UpdateDeferPhaseParams(ctx, tp, map[string]interface{}{"version": "0.75.0"})
	// make sure output artifact is set in DeferPhase
	c.Assert(tp.DeferPhase.Output, DeepEquals, map[string]interface{}{"version": "0.75.0"})
	c.Assert(tp.Phases, HasLen, 1)
	c.Assert(tp.Phases["backup"], NotNil)
	c.Assert(tp.Secrets, HasLen, 1)
	c.Assert(tp.Secrets["actionSetSecret"], DeepEquals, *secret)
}

func (s *ParamsSuite) TestRenderingPhaseParams(c *C) {
	ctx := context.Background()
	secret := &corev1.Secret{
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
		"authSecret": {
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
		t, err := template.New("config").Option("missingkey=error").Funcs(ksprig.TxtFuncMap()).Parse(tc.arg)
		c.Assert(err, IsNil)
		buf := bytes.NewBuffer(nil)
		err = t.Execute(buf, tp)
		c.Assert(err, IsNil)
		c.Assert(buf.String(), Equals, tc.expected)
	}
}

func newDeploymentConfig() *osapps.DeploymentConfig {
	return &osapps.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "tmp",
		},
		Spec: osapps.DeploymentConfigSpec{
			Replicas: 1,
			Selector: map[string]string{
				"app": "test",
			},
			Template: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image:   "alpine",
							Name:    "container",
							Command: []string{"tail", "-f", "/dev/null"},
						},
					},
				},
			},
		},
	}
}
