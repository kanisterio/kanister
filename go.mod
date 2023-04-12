module github.com/kanisterio/kanister

go 1.19

replace (
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.4.1
	github.com/graymeta/stow => github.com/kastenhq/stow v0.2.6-kasten.1.0.20220726203146-8a90401257d4
	github.com/rook/operator-kit => github.com/kastenhq/operator-kit v0.0.0-20180316185208-859e831cc18d
	gopkg.in/check.v1 => github.com/kastenhq/check v0.0.0-20180626002341-0264cfcea734
)

// Direct and indirect dependencies are grouped together
require (
	github.com/Azure/azure-sdk-for-go v68.0.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.28
	github.com/Azure/go-autorest/autorest/adal v0.9.23
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.12
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/BurntSushi/toml v1.2.1
	github.com/IBM/ibmcloud-storage-volume-lib v1.0.2-beta02.0.20190828145158-1da4543a60af
	github.com/Masterminds/semver v1.5.0
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/aws/aws-sdk-go v1.44.239
	github.com/dustin/go-humanize v1.0.1
	github.com/go-logr/logr v1.2.3
	github.com/go-openapi/strfmt v0.21.3
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/golang/mock v1.6.0
	github.com/google/uuid v1.3.0
	github.com/graymeta/stow v0.0.0-00010101000000-000000000000
	github.com/hashicorp/go-version v1.6.0
	github.com/jpillora/backoff v1.0.0
	github.com/json-iterator/go v1.1.12
	github.com/kopia/kopia v0.12.2-0.20230223181807-5c901abc9085
	github.com/kubernetes-csi/external-snapshotter/client/v4 v4.2.0
	github.com/lib/pq v1.10.7
	github.com/luci/go-render v0.0.0-20160219211803-9a04cc21af0f
	github.com/mitchellh/mapstructure v1.5.0

	//pinned openshift to release-4.5 branch
	github.com/openshift/api v0.0.0-20200526144822-34f54f12813a
	github.com/openshift/client-go v0.0.0-20200521150516-05eb9880269c
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.14.0
	github.com/sirupsen/logrus v1.9.0
	github.com/spf13/cobra v1.7.0
	github.com/vmware/govmomi v0.30.4
	go.uber.org/zap v1.24.0
	golang.org/x/oauth2 v0.7.0
	google.golang.org/api v0.117.0
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c
	gopkg.in/tomb.v2 v2.0.0-20161208151619-d5d1b5820637

	//pinned k8s.io to v0.26.x tag
	k8s.io/api v0.26.3
	k8s.io/apiextensions-apiserver v0.26.3
	k8s.io/apimachinery v0.26.3
	k8s.io/cli-runtime v0.26.3
	k8s.io/client-go v0.26.3
	k8s.io/kubectl v0.26.3
	k8s.io/utils v0.0.0-20230406110748-d93618cff8a2
	sigs.k8s.io/controller-runtime v0.14.6
	sigs.k8s.io/kustomize/kyaml v0.13.9
	sigs.k8s.io/yaml v1.3.0
)

require (
	github.com/Azure/azure-sdk-for-go/sdk/azcore v0.22.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v0.9.1 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/IBM-Cloud/ibm-cloud-cli-sdk v0.3.0 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/elazarl/goproxy v0.0.0-20190711103511-473e67f1d7d2 // indirect
	github.com/elazarl/goproxy/ext v0.0.0-20221015165544-a0805db90819 // indirect
	github.com/fatih/structs v1.1.0 // indirect
	github.com/go-errors/errors v1.4.2 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/jarcoal/httpmock v1.0.4 // indirect
	github.com/kelseyhightower/envconfig v1.4.0 // indirect
	github.com/mitchellh/copystructure v1.0.0 // indirect
	github.com/renier/xmlrpc v0.0.0-20170708154548-ce4a1a486c03 // indirect
	github.com/rogpeppe/go-internal v1.10.0 // indirect
	github.com/softlayer/softlayer-go v0.0.0-20190615201252-ba6e7f295217 // indirect
	go.mongodb.org/mongo-driver v1.10.1 // indirect
	k8s.io/kube-openapi v0.0.0-20230109183929-3758b55a6596 // indirect
)
