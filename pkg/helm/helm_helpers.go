package helm

import (
	"regexp"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
)

type k8sObj struct {
	ObjKind  string            `json:"kind"`
	MetaData metav1.ObjectMeta `json:"metadata"`
}

type K8sObjectType string

type Component struct {
	k8sType K8sObjectType
	name    string
}

func ParseReleaseNameFromHelmStatus(helmStatus string) string {
	re := regexp.MustCompile(`.*NAME:\s+(.*)\n`)
	withNameRE := regexp.MustCompile(`^Release\s+"(.*)"\s+`)
	tmpRelease := re.FindAllStringSubmatch(helmStatus, -1)
	log.Info().Print("Parsed output for generate name install")
	if len(tmpRelease) < 1 {
		tmpRelease = withNameRE.FindAllStringSubmatch(helmStatus, -1)
		log.Info().Print("Parsed output for specified name install/upgrade")
		if len(tmpRelease) < 1 {
			return ""
		}
	}
	if len(tmpRelease[0]) == 2 {
		return tmpRelease[0][1]
	}
	return ""
}

func ComponentsFromManifest(manifest string) []Component {
	var ret []Component
	for _, objYaml := range strings.Split(manifest, "---") {
		tmpK8s := k8sObj{}
		if err := yaml.Unmarshal([]byte(objYaml), &tmpK8s); err != nil {
			log.Error().Print("failed to Unmarshal k8s obj", field.M{"Error": err})
			continue
		}
		ret = append(ret, Component{k8sType: K8sObjectType(strings.ToLower(tmpK8s.ObjKind)), name: tmpK8s.MetaData.Name})
	}
	return ret
}
