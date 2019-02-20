package common

import (
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"io/ioutil"
	"path"
	"strings"
)

const (
	YummyAgentConfigPath = "/etc/yummy/config/"
)

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
