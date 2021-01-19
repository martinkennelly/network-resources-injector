package client

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var clientset kubernetes.Interface

func GetInClusterClient() (kubernetes.Interface, error) {
	if clientset == nil {
		if err := setupInClusterClient(); err != nil {
			return nil, err
		}
	}
	return clientset, nil
}

// setupInClusterClient sets package variable to enable communication to kubernetes API
func setupInClusterClient() error {
	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	return nil
}
