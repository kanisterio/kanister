package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
)

type CRV1alpha1Interface interface {
	RESTClient() rest.Interface
	ActionSetsGetter
	BlueprintsGetter
}

// CRV1alpha1Client is used to interact with features provided by the apps group.
type CRV1alpha1Client struct {
	restClient rest.Interface
	scheme     *runtime.Scheme
}

func (c *CRV1alpha1Client) ActionSets(namespace string) ActionSetInterface {
	return newActionSets(c, namespace, runtime.NewParameterCodec(c.scheme))
}

func (c *CRV1alpha1Client) Blueprints(namespace string) BlueprintInterface {
	return newBlueprints(c, namespace, runtime.NewParameterCodec(c.scheme))
}

// NewForConfig creates a new CRV1alpha1Client for the given config.
func NewForConfig(c *rest.Config) (*CRV1alpha1Client, error) {
	config := *c
	scheme := runtime.NewScheme()
	if err := SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	setConfigDefaults(&config, scheme)
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &CRV1alpha1Client{client, scheme}, nil
}

// NewForConfigOrDie creates a new CRV1alpha1Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *CRV1alpha1Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

func setConfigDefaults(config *rest.Config, scheme *runtime.Scheme) {
	config.GroupVersion = &SchemeGroupVersion
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: serializer.NewCodecFactory(scheme)}

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *CRV1alpha1Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
