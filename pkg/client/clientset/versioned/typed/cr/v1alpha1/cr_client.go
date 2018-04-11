package v1alpha1

import (
	v1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/client/clientset/versioned/scheme"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer"
	rest "k8s.io/client-go/rest"
)

type CrV1alpha1Interface interface {
	RESTClient() rest.Interface
	ActionSetsGetter
	BlueprintsGetter
	ProfilesGetter
}

// CrV1alpha1Client is used to interact with features provided by the cr group.
type CrV1alpha1Client struct {
	restClient rest.Interface
}

func (c *CrV1alpha1Client) ActionSets(namespace string) ActionSetInterface {
	return newActionSets(c, namespace)
}

func (c *CrV1alpha1Client) Blueprints(namespace string) BlueprintInterface {
	return newBlueprints(c, namespace)
}

func (c *CrV1alpha1Client) Profiles(namespace string) ProfileInterface {
	return newProfiles(c, namespace)
}

// NewForConfig creates a new CrV1alpha1Client for the given config.
func NewForConfig(c *rest.Config) (*CrV1alpha1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &CrV1alpha1Client{client}, nil
}

// NewForConfigOrDie creates a new CrV1alpha1Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *CrV1alpha1Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new CrV1alpha1Client for the given RESTClient.
func New(c rest.Interface) *CrV1alpha1Client {
	return &CrV1alpha1Client{c}
}

func setConfigDefaults(config *rest.Config) error {
	gv := v1alpha1.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *CrV1alpha1Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
