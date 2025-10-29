package k8s

import (
	"log"
	"os"
	"path/filepath"
	"sync"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	clientset  *kubernetes.Clientset
	clientInit sync.Once
)

// GetClient lazily initializes it on first use.
func GetClient() *kubernetes.Clientset {
	clientInit.Do(func() {
		log.Printf("Getting Kubernetes Client")
		var config *rest.Config
		var err error

		// 1. Try in-cluster config first
		config, err = rest.InClusterConfig()
		if err != nil {
			log.Printf("Couldn't get in cluster config: falling back to kubeconfig: %v", err)
			// If not running in cluster, fallback to kubeconfig
			kubeconfig := filepath.Join(
				os.Getenv("HOME"), ".kube", "config",
			)
			config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
			if err != nil {
				log.Fatalf("failed to load kubeconfig: %v", err)
			}
		}

		cs, err := kubernetes.NewForConfig(config)
		if err != nil {
			log.Fatalf("Failed to create Kubernetes client: %v", err)
		}

		clientset = cs
	})
	return clientset
}
