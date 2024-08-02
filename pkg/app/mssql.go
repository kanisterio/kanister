package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kanisterio/errkit"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
)

const (
	mssqlWaitTimeout = 5 * time.Minute
	dbUserName       = "sa"
	dbPass           = "MyC0m9l&xP@ssw0rd"
	connString       = "/opt/mssql-tools/bin/sqlcmd -S localhost -U %s -P \"%s\" -Q "
)

type MssqlDB struct {
	cli        kubernetes.Interface
	namespace  string
	name       string
	deployment *appsv1.Deployment
	service    *corev1.Service
	pvc        *corev1.PersistentVolumeClaim
	secret     *corev1.Secret
}

func NewMssqlDB(name string) App {
	return &MssqlDB{
		name: name,
	}
}

func (m *MssqlDB) ConfigMaps() map[string]crv1alpha1.ObjectReference {
	return nil
}

func (m *MssqlDB) Secrets() map[string]crv1alpha1.ObjectReference {
	return map[string]crv1alpha1.ObjectReference{
		"mssql": {
			Kind:      "Secret",
			Name:      m.name,
			Namespace: m.namespace,
		},
	}
}

func (m *MssqlDB) Init(ctx context.Context) error {
	cfg, err := kube.LoadConfig()
	if err != nil {
		return err
	}

	m.cli, err = kubernetes.NewForConfig(cfg)
	return err
}

func (m *MssqlDB) Install(ctx context.Context, namespace string) error {
	m.namespace = namespace
	secret, err := m.cli.CoreV1().Secrets(namespace).Create(ctx, m.getSecretObj(), metav1.CreateOptions{})
	if err != nil {
		return err
	}
	log.Print("Secret created successfully", field.M{"app": m.name, "secret": secret.Name})
	m.secret = secret

	pvcObj, err := m.getPVCObj()
	if err != nil {
		return err
	}
	pvc, err := m.cli.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, pvcObj, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	log.Print("PVC created successfully", field.M{"app": m.name, "pvc": pvc.Name})
	m.pvc = pvc

	deploymentObj, err := m.getDeploymentObj()
	if err != nil {
		return err
	}
	deployment, err := m.cli.AppsV1().Deployments(namespace).Create(ctx, deploymentObj, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	log.Print("Deployment created successfully", field.M{"app": m.name, "deployment": deployment.Name})
	m.deployment = deployment

	serviceObj, err := m.getServiceObj()
	if err != nil {
		return err
	}
	service, err := m.cli.CoreV1().Services(namespace).Create(ctx, serviceObj, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	log.Print("Service created successfully", field.M{"app": m.name, "service": service.Name})
	m.service = service

	return nil
}

func (m *MssqlDB) IsReady(ctx context.Context) (bool, error) {
	log.Print("Waiting for the mssql application to be ready.", field.M{"app": m.name})
	ctx, cancel := context.WithTimeout(ctx, mssqlWaitTimeout)
	defer cancel()

	err := kube.WaitOnDeploymentReady(ctx, m.cli, m.namespace, m.deployment.Name)
	if err != nil {
		return false, err
	}
	log.Print("Application instance is ready.", field.M{"app": m.name})
	return true, nil
}

func (m *MssqlDB) Object() crv1alpha1.ObjectReference {
	return crv1alpha1.ObjectReference{
		Kind:      "deployment",
		Name:      "mssql-deployment",
		Namespace: m.namespace,
	}
}

func (m *MssqlDB) Uninstall(ctx context.Context) error {
	err := m.cli.AppsV1().Deployments(m.namespace).Delete(ctx, m.deployment.Name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	log.Print("Deployment deleted successfully", field.M{"app": m.name})

	err = m.cli.CoreV1().PersistentVolumeClaims(m.namespace).Delete(ctx, m.pvc.Name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	log.Print("PVC deleted successfully", field.M{"app": m.name})

	err = m.cli.CoreV1().Services(m.namespace).Delete(ctx, m.service.Name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	log.Print("Service deleted successfully", field.M{"app": m.name})

	err = m.cli.CoreV1().Secrets(m.namespace).Delete(ctx, m.secret.Name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	log.Print("Secret deleted successfully", field.M{"app": m.name})
	return nil
}

func (m *MssqlDB) Ping(ctx context.Context) error {
	log.Print("Pinging mssql database", field.M{"app": m.name})
	count := fmt.Sprintf(connString+
		"\"SELECT name FROM sys.databases WHERE name NOT IN ('master','model','msdb','tempdb')\" -b -s \",\" -h -1", dbUserName, dbPass)

	loginMssql := []string{"sh", "-c", count}
	_, stderr, err := m.execCommand(ctx, loginMssql)
	if err != nil {
		return errkit.Wrap(err, "Error while Pinging the database", "stderr", stderr)
	}
	log.Print("Ping to the application was success.", field.M{"app": m.name})
	return nil
}

func (m *MssqlDB) Insert(ctx context.Context) error {
	log.Print("Adding entry to database", field.M{"app": m.name})
	insert := fmt.Sprintf(connString+
		"\"USE test; INSERT INTO Inventory VALUES (1, 'banana', 150)\"", dbUserName, dbPass)

	insertQuery := []string{"sh", "-c", insert}
	_, stderr, err := m.execCommand(ctx, insertQuery)
	if err != nil {
		return errkit.Wrap(err, "Error while inserting data into table", "stderr", stderr)
	}
	return nil
}

func (m *MssqlDB) Count(ctx context.Context) (int, error) {
	log.Print("Counting entries from database", field.M{"app": m.name})
	insert := fmt.Sprintf(connString+
		"\"SET NOCOUNT ON; USE test; SELECT COUNT(*) FROM Inventory\" -h -1", dbUserName, dbPass)

	insertQuery := []string{"sh", "-c", insert}
	stdout, stderr, err := m.execCommand(ctx, insertQuery)
	if err != nil {
		return 0, errkit.Wrap(err, "Error while inserting data into table", "stderr", stderr)
	}
	rowsReturned, err := strconv.Atoi(strings.TrimSpace(strings.Split(stdout, "\n")[1]))
	if err != nil {
		return 0, errkit.Wrap(err, "Error while converting response of count query", "stderr", stderr)
	}
	return rowsReturned, nil
}

func (m *MssqlDB) Reset(ctx context.Context) error {
	log.Print("Reseting database", field.M{"app": m.name})
	delete := fmt.Sprintf(connString+"\"DROP DATABASE test\"", dbUserName, dbPass)
	deleteQuery := []string{"sh", "-c", delete}
	_, stderr, err := m.execCommand(ctx, deleteQuery)
	if err != nil {
		return errkit.Wrap(err, "Error while inserting data into table", "stderr", stderr)
	}
	return nil
}

func (m *MssqlDB) Initialize(ctx context.Context) error {
	log.Print("Initializing database", field.M{"app": m.name})
	createDB := fmt.Sprintf(connString+"\"CREATE DATABASE test\"", dbUserName, dbPass)

	createTable := fmt.Sprintf(connString+
		"\"USE test; CREATE TABLE Inventory (id INT, name NVARCHAR(50), quantity INT)\"", dbUserName, dbPass)

	execQuery := []string{"sh", "-c", createDB}
	_, stderr, err := m.execCommand(ctx, execQuery)
	if err != nil {
		return errkit.Wrap(err, "Error while creating the database", "stderr", stderr)
	}

	execQuery = []string{"sh", "-c", createTable}
	_, stderr, err = m.execCommand(ctx, execQuery)
	if err != nil {
		return errkit.Wrap(err, "Error while creating table", "stderr", stderr)
	}
	return nil
}

func (m *MssqlDB) GetClusterScopedResources(ctx context.Context) []crv1alpha1.ObjectReference {
	return nil
}

func (m MssqlDB) execCommand(ctx context.Context, command []string) (string, string, error) {
	podName, containerName, err := kube.GetPodContainerFromDeployment(ctx, m.cli, m.namespace, m.deployment.Name)
	if err != nil || podName == "" {
		return "", "", errkit.Wrap(err, "Error getting pod and container name for app.", "app", m.name)
	}
	return kube.Exec(ctx, m.cli, m.namespace, podName, containerName, command, nil)
}

func (m *MssqlDB) getDeploymentObj() (*appsv1.Deployment, error) {
	deploymentManifest :=
		`apiVersion: apps/v1
kind: Deployment
metadata:
  name: mssql-deployment
  labels:
    app: mssql
spec:
  replicas: 1
  selector:
    matchLabels:
      app: mssql
  template:
    metadata:
      labels:
        app: mssql
    spec:
      terminationGracePeriodSeconds: 30
      hostname: mssqlinst
      securityContext:
        fsGroup: 10001
      containers:
        - name: mssql
          image: mcr.microsoft.com/mssql/server:2019-CU27-ubuntu-20.04
          ports:
            - containerPort: 1433
          env:
            - name: MSSQL_PID
              value: "Developer"
            - name: ACCEPT_EULA
              value: "Y"
            - name: SA_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: mssql
                  key: SA_PASSWORD
          volumeMounts:
            - name: mssqldb
              mountPath: /var/opt/mssql
      volumes:
        - name: mssqldb
          persistentVolumeClaim:
            claimName: mssql-data`

	var deployment *appsv1.Deployment
	err := yaml.Unmarshal([]byte(deploymentManifest), &deployment)
	return deployment, err
}

func (m *MssqlDB) getPVCObj() (*corev1.PersistentVolumeClaim, error) {
	pvcmaniFest :=
		`kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: mssql-data
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 4Gi`

	var pvc *corev1.PersistentVolumeClaim
	err := yaml.Unmarshal([]byte(pvcmaniFest), &pvc)
	return pvc, err
}

func (m *MssqlDB) getServiceObj() (*corev1.Service, error) {
	serviceManifest :=
		`apiVersion: v1
kind: Service
metadata:
  name: mssql-deployment
spec:
  selector:
    app: mssql
  ports:
    - protocol: TCP
      port: 1433
      targetPort: 1433
  type: ClusterIP`

	var service *corev1.Service
	err := yaml.Unmarshal([]byte(serviceManifest), &service)
	return service, err
}

func (m MssqlDB) getSecretObj() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: m.name,
		},
		Data: map[string][]byte{
			"SA_PASSWORD": []byte(dbPass),
		},
		Type: "Opaque",
	}
}
