package app

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
)

const (
	mssqlWaitTimeout = 2 * time.Minute
)

type MssqlDB struct {
	cli        kubernetes.Interface
	namespace  string
	name       string
	deployment *appsv1.Deployment
	service    *v1.Service
	pvc        *v1.PersistentVolumeClaim
	secret     *v1.Secret
	dbUserName string
	dbPass     string
}

func NewMssqlDB(name string) App {
	return &MssqlDB{
		name: name,
		// These values are hard coded creating secret
		dbUserName: "sa",
		dbPass:     "MyC0m9l&xP@ssw0rd",
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
	log.Print("Secret with name "+secret.Name+" created successfully", field.M{"app": m.name})
	m.secret = secret

	pvcObj, err := m.getPVCObj()
	if err != nil {
		return err
	}
	pvc, err := m.cli.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, pvcObj, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	log.Print("PVC with name "+pvc.Name+" created successfully", field.M{"app": m.name})
	m.pvc = pvc

	deploymentObj, err := m.getDeploymentObj()
	if err != nil {
		return err
	}
	deployment, err := m.cli.AppsV1().Deployments(namespace).Create(ctx, deploymentObj, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	log.Print("Deployment with name "+deployment.Name+" created successfully", field.M{"app": m.name})
	m.deployment = deployment

	serviceObj, err := m.getServiceObj()
	if err != nil {
		return err
	}
	service, err := m.cli.CoreV1().Services(namespace).Create(ctx, serviceObj, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	log.Print("Service with name "+service.Name+" created successfully", field.M{"app": m.name})
	m.service = service

	return err
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
	log.Debug().Print("Deployment deleted successfully", field.M{"app": m.name})

	err = m.cli.CoreV1().PersistentVolumeClaims(m.namespace).Delete(ctx, m.pvc.Name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	log.Debug().Print("PVC deleted successfully", field.M{"app": m.name})

	err = m.cli.CoreV1().Services(m.namespace).Delete(ctx, m.service.Name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	log.Debug().Print("Service deleted successfully", field.M{"app": m.name})

	err = m.cli.CoreV1().Secrets(m.namespace).Delete(ctx, m.secret.Name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	log.Debug().Print("Secret deleted successfully", field.M{"app": m.name})
	return nil
}

func (m *MssqlDB) Ping(ctx context.Context) error {
	log.Print("Pinging mssql database", field.M{"app": m.name})
	count := "/opt/mssql-tools/bin/sqlcmd -S localhost -U " + m.dbUserName + " -P \"" + m.dbPass + "\" -Q " +
		"\"SELECT name FROM sys.databases WHERE name NOT IN ('master','model','msdb','tempdb')\" -b -s \",\" -h -1"

	loginMssql := []string{"sh", "-c", count}
	_, stderr, err := m.execCommand(ctx, loginMssql)
	if err != nil {
		return errors.Wrapf(err, "Error while Pinging the database: %s", stderr)
	}
	log.Print("Ping to the application was success.", field.M{"app": m.name})
	return err
}

func (m *MssqlDB) Insert(ctx context.Context) error {
	log.Print("Adding entry to database", field.M{"app": m.name})
	insert := "/opt/mssql-tools/bin/sqlcmd -S localhost -U " + m.dbUserName + " -P \"" + m.dbPass + "\" -Q " +
		"\"USE test; INSERT INTO Inventory VALUES (1, 'banana', 150)\""

	insertQuery := []string{"sh", "-c", insert}
	_, stderr, err := m.execCommand(ctx, insertQuery)
	if err != nil {
		return errors.Wrapf(err, "Error while inserting data into table: %s", stderr)
	}
	return err
}

func (m *MssqlDB) Count(ctx context.Context) (int, error) {
	log.Print("Counting entries from database", field.M{"app": m.name})
	insert := "/opt/mssql-tools/bin/sqlcmd -S localhost -U " + m.dbUserName + " -P \"" + m.dbPass + "\" -Q " +
		"\"SET NOCOUNT ON; USE test; SELECT COUNT(*) FROM Inventory\" -h -1"

	insertQuery := []string{"sh", "-c", insert}
	stdout, stderr, err := m.execCommand(ctx, insertQuery)
	if err != nil {
		return 0, errors.Wrapf(err, "Error while inserting data into table: %s", stderr)
	}
	rowsReturned, err := strconv.Atoi(strings.TrimSpace(strings.Split(stdout, "\n")[1]))
	if err != nil {
		return 0, errors.Wrapf(err, "Error while converting data: %s", stderr)
	}
	return rowsReturned, nil
}

func (m *MssqlDB) Reset(ctx context.Context) error {
	log.Print("Reseting database", field.M{"app": m.name})
	delete := "/opt/mssql-tools/bin/sqlcmd -S localhost -U " + m.dbUserName + " -P \"" + m.dbPass + "\" -Q " +
		"\"DROP DATABASE test\""
	deleteQuery := []string{"sh", "-c", delete}
	_, stderr, err := m.execCommand(ctx, deleteQuery)
	if err != nil {
		return errors.Wrapf(err, "Error while inserting data into table: %s", stderr)
	}
	return err
}

func (m *MssqlDB) Initialize(ctx context.Context) error {
	log.Print("Initializing database", field.M{"app": m.name})
	createDB := "/opt/mssql-tools/bin/sqlcmd -S localhost -U " + m.dbUserName + " -P \"" + m.dbPass + "\" -Q " +
		"\"CREATE DATABASE test\""

	createTable := "/opt/mssql-tools/bin/sqlcmd -S localhost -U " + m.dbUserName + " -P \"" + m.dbPass + "\" -Q " +
		"\"USE test; CREATE TABLE Inventory (id INT, name NVARCHAR(50), quantity INT)\""

	execQuery := []string{"sh", "-c", createDB}
	_, stderr, err := m.execCommand(ctx, execQuery)
	if err != nil {
		return errors.Wrapf(err, "Error while creating the database: %s", stderr)
	}

	execQuery = []string{"sh", "-c", createTable}
	_, stderr, err = m.execCommand(ctx, execQuery)
	if err != nil {
		return errors.Wrapf(err, "Error while creating table: %s", stderr)
	}
	return err
}

func (m *MssqlDB) GetClusterScopedResources(ctx context.Context) []crv1alpha1.ObjectReference {
	return nil
}

func (m MssqlDB) execCommand(ctx context.Context, command []string) (string, string, error) {
	podName, containerName, err := kube.GetPodContainerFromDeployment(ctx, m.cli, m.namespace, m.deployment.Name)
	if err != nil || podName == "" {
		return "", "", errors.Wrapf(err, "Error  getting pod and containername %s.", m.name)
	}
	return kube.Exec(m.cli, m.namespace, podName, containerName, command, nil)
}
