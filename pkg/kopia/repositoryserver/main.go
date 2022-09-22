package main

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	rsclient "github.com/kanisterio/kanister/pkg/kopia/repositoryserver/pkg/client/clientset/versioned"
	rsinformers "github.com/kanisterio/kanister/pkg/kopia/repositoryserver/pkg/client/informers/externalversions"
	"github.com/kanisterio/kanister/pkg/kopia/repositoryserver/pkg/controller"
)

func main() {
	config, err := rest.InClusterConfig()
	if err != nil {
		// fallback to kubeconfig
		home := homedir.HomeDir()
		kubeconfig := filepath.Join(home, ".kube", "config")
		if envvar := os.Getenv("KUBECONFIG"); len(envvar) > 0 {
			kubeconfig = envvar
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			log.Printf("Building config from flags failed %s", err.Error())
			os.Exit(1)
		}
	}
	rsclientset, err := rsclient.NewForConfig(config)
	if err != nil {
		log.Println("Error in fetching the RepositoryServer clientset", err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Printf("Error in getting standard clientset %s\n", err.Error())
	}

	infoFactory := rsinformers.NewSharedInformerFactory(rsclientset, 20*time.Minute)
	ch := make(chan struct{})
	c := controller.NewController(clientset, rsclientset, infoFactory.Cr().V1alpha1().RepositoryServers())

	infoFactory.Start(ch)
	if err := c.Run(ch); err != nil {
		log.Printf("Error running controller %s\n", err.Error())
	}
}
