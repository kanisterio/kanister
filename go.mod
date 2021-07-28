module github.com/kanisterio/kanister

go 1.12

replace (
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.4.1
	github.com/graymeta/stow => github.com/kastenhq/stow v0.2.6-kasten.1
	github.com/rook/operator-kit => github.com/kastenhq/operator-kit v0.0.0-20180316185208-859e831cc18d
	gopkg.in/check.v1 => github.com/kastenhq/check v0.0.0-20180626002341-0264cfcea734
)

require (
	cloud.google.com/go v0.88.0 // indirect
	cloud.google.com/go/storage v1.16.0 // indirect
	github.com/Azure/azure-sdk-for-go v54.0.0+incompatible
	github.com/Azure/azure-storage-blob-go v0.14.0 // indirect
	github.com/Azure/go-autorest/autorest v0.11.19
	github.com/Azure/go-autorest/autorest/adal v0.9.14 // indirect
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.7
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/BurntSushi/toml v0.3.1
	github.com/IBM-Cloud/ibm-cloud-cli-sdk v0.3.0 // indirect
	github.com/IBM/ibmcloud-storage-volume-lib v1.0.2-beta02.0.20190828145158-1da4543a60af
	github.com/Masterminds/semver v1.4.2
	github.com/Masterminds/sprig v2.15.0+incompatible
	github.com/aokoli/goutils v1.1.0 // indirect
	github.com/asaskevich/govalidator v0.0.0-20200907205600-7a23bdc65eef // indirect
	github.com/aws/aws-sdk-go v1.38.69
	github.com/dnaeon/go-vcr v1.0.1 // indirect
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/efarrer/iothrottler v0.0.2 // indirect
	github.com/elazarl/goproxy v0.0.0-20190711103511-473e67f1d7d2 // indirect
	github.com/elazarl/goproxy/ext v0.0.0-20190711103511-473e67f1d7d2 // indirect
	github.com/form3tech-oss/jwt-go v3.2.3+incompatible // indirect
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-openapi/strfmt v0.19.3
	github.com/go-sql-driver/mysql v1.6.0
	github.com/gofrs/uuid v4.0.0+incompatible // indirect
	github.com/golang/mock v1.6.0
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.3.0
	github.com/googleapis/gnostic v0.5.3 // indirect
	github.com/graymeta/stow v0.0.0-00010101000000-000000000000
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/imdario/mergo v0.3.11 // indirect
	github.com/jarcoal/httpmock v1.0.4 // indirect
	github.com/jpillora/backoff v1.0.0
	github.com/json-iterator/go v1.1.11
	github.com/kelseyhightower/envconfig v1.4.0 // indirect
	github.com/klauspost/compress v1.13.1 // indirect
	github.com/klauspost/cpuid/v2 v2.0.8 // indirect
	github.com/kopia/kopia v0.8.2-0.20210706000840-5642a8a52192
	github.com/kubernetes-csi/external-snapshotter/client/v4 v4.0.0
	github.com/lib/pq v1.10.1
	github.com/luci/go-render v0.0.0-20160219211803-9a04cc21af0f
	github.com/minio/minio-go/v7 v7.0.12 // indirect
	github.com/mitchellh/mapstructure v1.4.1
	github.com/natefinch/atomic v1.0.1 // indirect

	//pinned openshift to release-4.5 branch
	github.com/openshift/api v0.0.0-20200526144822-34f54f12813a
	github.com/openshift/client-go v0.0.0-20200521150516-05eb9880269c
	github.com/pierrec/lz4 v2.6.1+incompatible // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/common v0.30.0 // indirect
	github.com/prometheus/procfs v0.7.1 // indirect
	github.com/renier/xmlrpc v0.0.0-20170708154548-ce4a1a486c03 // indirect
	github.com/rs/xid v1.3.0 // indirect
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.8.1
	github.com/softlayer/softlayer-go v0.0.0-20190615201252-ba6e7f295217 // indirect
	github.com/spf13/cobra v1.1.1
	github.com/vmware/govmomi v0.21.1-0.20191008161538-40aebf13ba45
	github.com/zeebo/blake3 v0.1.2 // indirect
	go.mongodb.org/mongo-driver v1.5.1 // indirect
	go.uber.org/zap v1.16.0
	golang.org/x/crypto v0.0.0-20210711020723-a769d52b0f97 // indirect
	golang.org/x/exp v0.0.0-20210722180016-6781d3edade3 // indirect
	golang.org/x/net v0.0.0-20210726213435-c6fcb2dbf985 // indirect
	golang.org/x/oauth2 v0.0.0-20210628180205-a41e5a781914
	golang.org/x/term v0.0.0-20210615171337-6886f2dfbf5b // indirect
	google.golang.org/api v0.51.0
	google.golang.org/genproto v0.0.0-20210726200206-e7812ac95cc0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c
	gopkg.in/tomb.v2 v2.0.0-20161208151619-d5d1b5820637

	//pinned k8s.io to v0.20.1 tag
	k8s.io/api v0.20.1
	k8s.io/apiextensions-apiserver v0.20.1
	k8s.io/apimachinery v0.20.1
	k8s.io/client-go v0.20.1
)
