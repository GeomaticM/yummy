package utils

import (
	"github.com/golang/glog"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"os/exec"
)

const (
	KubeConfigEnv = "KUBECONFIG"
)

// BuildConfigFromFlags being defined to enable mocking during unit testing
var BuildConfigFromFlags = clientcmd.BuildConfigFromFlags

// InClusterConfig being defined to enable mocking during unit testing
var InClusterConfig = rest.InClusterConfig

func Run(cmd string, args ...string) error {
	glog.Infof("Running %s %s", cmd, args)
	out, err := exec.Command(cmd, args...).CombinedOutput()
	if err != nil {
		glog.Errorf("Error running %s %v: %v, %s", cmd, args, err, out)
	}
	return err
}

// SetupClient created client using either in-cluster configuration or if KUBECONFIG environment variable is specified then using that config.
func SetupClient() *kubernetes.Clientset {
	var config *rest.Config
	var err error

	kubeconfigFile := os.Getenv(KubeConfigEnv)
	if kubeconfigFile != "" {
		config, err = BuildConfigFromFlags("", kubeconfigFile)
		if err != nil {
			glog.Fatalf("Error creating config from %s specified file: %s %v\n", KubeConfigEnv,
				kubeconfigFile, err)
		}
		glog.Infof("Creating client using kubeconfig file %s", kubeconfigFile)
	} else {
		config, err = InClusterConfig()
		if err != nil {
			glog.Fatalf("Error creating InCluster config: %v\n", err)
		}
		glog.Infof("Creating client using in-cluster config")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatalf("Error creating clientset: %v\n", err)
	}
	return clientset
}
