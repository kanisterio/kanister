package app

import (
	"github.com/ghodss/yaml"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	dbUserName = "sa"
	dbPass     = "MyC0m9l&xP@ssw0rd"
)

func (m *MssqlDB) getDeploymentObj() (*appsv1.Deployment, error) {
	deploymentManifest :=
		`apiVersion: apps/v1
kind: Deployment
metadata:
  name: mssql-deployment
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
          image: mcr.microsoft.com/mssql/server:2019-latest
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

func (m *MssqlDB) getPVCObj() (*v1.PersistentVolumeClaim, error) {
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

	var pvc *v1.PersistentVolumeClaim
	err := yaml.Unmarshal([]byte(pvcmaniFest), &pvc)
	return pvc, err
}

func (m *MssqlDB) getServiceObj() (*v1.Service, error) {
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

	var service *v1.Service
	err := yaml.Unmarshal([]byte(serviceManifest), &service)
	return service, err

}

func (m MssqlDB) getSecretObj() *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: m.name,
		},
		Data: map[string][]byte{
			"SA_PASSWORD": []byte(dbPass),
		},
		Type: "Opaque",
	}
}
