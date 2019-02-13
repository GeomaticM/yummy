package common

import (
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path"
	"strings"
)

const (
	YummyAgentConfigPath = "/etc/yummy/config/"
	KubeConfigEnv        = "KUBECONFIG"
)

// BuildConfigFromFlags being defined to enable mocking during unit testing
var BuildConfigFromFlags = clientcmd.BuildConfigFromFlags

// InClusterConfig being defined to enable mocking during unit testing
var InClusterConfig = rest.InClusterConfig

type AgentConfig struct {
	// The volume group
	VolumeGroup string `json:"volumeGroup" yaml:"volumeGroup"`
	MountDir    string `json:"mountDir" yaml:"mountDir"`
}

// YummyAgentConfig stores a configuration for yummy agent
type YummyAgentConfig struct {
	AgentConfigMap map[string]AgentConfig `json:"agentConfigMap" yaml:"agentConfigMap"`
}

func insertSpaces(original string) string {
	spaced := ""
	for _, line := range strings.Split(original, "\n") {
		spaced += "   "
		spaced += line
		spaced += "\n"
	}
	return spaced
}

// ConfigMapDataToVolumeConfig converts configmap data to volume config.
func ConfigMapDataToVolumeConfig(data map[string]string, yummyAgentConfig *YummyAgentConfig) error {
	rawYaml := ""
	for key, val := range data {
		rawYaml += key
		rawYaml += ": \n"
		rawYaml += insertSpaces(string(val))
	}

	if err := yaml.Unmarshal([]byte(rawYaml), yummyAgentConfig); err != nil {
		return fmt.Errorf("fail to Unmarshal yaml due to: %#v", err)
	}
	for class, config := range yummyAgentConfig.AgentConfigMap {
		if config.VolumeGroup == "" {
			return fmt.Errorf("yummy agent %v is misconfigured, missing volume group parameter", class)
		}

		yummyAgentConfig.AgentConfigMap[class] = config
		glog.Infof("StorageClass %q configured with MountDir %q, HostDir %q, VolumeMode %q, FsType %q, BlockCleanerCommand %q",
			class,
			config.VolumeGroup,
		)
	}
	return nil
}

func LoadYummyAgentConfigs(configPath string, yummyAgentConfig *YummyAgentConfig) error {
	files, err := ioutil.ReadDir(configPath)
	if err != nil {
		return err
	}
	data := make(map[string]string)
	for _, file := range files {
		if !file.IsDir() {
			if strings.Compare(file.Name(), "..data") != 0 {
				fileContents, err := ioutil.ReadFile(path.Join(configPath, file.Name()))
				if err != nil {
					glog.Infof("Could not read file: %s due to: %v", path.Join(configPath, file.Name()), err)
					return err
				}
				data[file.Name()] = string(fileContents)
			}
		}
	}
	return ConfigMapDataToVolumeConfig(data, yummyAgentConfig)
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
