package client

import (
	"math"
	"os"

	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/flowcontrol"
	"k8s.io/klog/v2"
)

// GetKubeConfig gets a kubeconfig object from the supplied file path.
func GetKubeConfig(kubeconfig string) *restclient.Config {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		klog.Errorf("Error reading the kubeconfig file: %v", err)
		os.Exit(1)
	}
	// Disable client-go rate-limiting, we'll manage the test throughput ourselves.
	config.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(math.MaxFloat32, math.MaxInt)
	return config
}

// CreateKubeClients creates a given number of k8s clients using the provided kubeconfig.
func CreateKubeClients(config *restclient.Config, numClients int) []*kubernetes.Clientset {
	clients := make([]*kubernetes.Clientset, 0, numClients)
	for i := 0; i < numClients; i++ {
		client, err := kubernetes.NewForConfig(config)
		if err != nil {
			klog.Errorf("Error creating a k8s client: %v", err)
			os.Exit(1)
		}
		clients = append(clients, client)
	}
	return clients
}
