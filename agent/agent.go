package main

import "C"

import (
	"flag"
	"github.com/golang/glog"
	"github.com/silenceshell/yummy/pkg/common"
	"github.com/silenceshell/yummy/pkg/controller"
	"github.com/silenceshell/yummy/pkg/lvm"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"math/rand"
	"os"
	"time"
)

var (
	optListenAddress string
	optMetricsPath   string
)

func main() {
	// init
	//   vg name as a param
	//   get lvm vg(vgdisplay) period: VG Size, Alloc, Free
	rand.Seed(time.Now().UTC().UnixNano())
	flag.StringVar(&optListenAddress, "listen-address", ":8080", "address on which to expose metrics")
	flag.StringVar(&optMetricsPath, "metrics-path", "/metrics", "path under which to expose metrics")
	flag.Set("logtostderr", "true")
	flag.Parse()

	yummyAgentConfig := common.YummyAgentConfig{
		AgentConfigMap: make(map[string]common.AgentConfig),
	}

	if err := common.LoadYummyAgentConfigs(common.YummyAgentConfigPath, &yummyAgentConfig); err != nil {
		glog.Fatalf("Error parsing Yummy's configuration: %#v. Exiting...\n", err)
	}
	glog.Infof("Loaded configuration: %+v", yummyAgentConfig)
	glog.Infof("Ready to run...")

	nodeName := os.Getenv("MY_NODE_NAME")
	if nodeName == "" {
		glog.Fatalf("MY_NODE_NAME environment variable not set\n")
	}

	namespace := os.Getenv("MY_NAMESPACE")
	if namespace == "" {
		glog.Warningf("MY_NAMESPACE environment variable not set, will be set to default.\n")
		namespace = "default"
	}

	client := common.SetupClient()
	node := getNode(client, nodeName)

	glog.Info("node name is ", node.Name)

	vg := yummyAgentConfig.AgentConfigMap["agentConfigMap"].VolumeGroup
	isAvailable, err := lvm.IsVgExistAndAvailable(vg)
	if err != nil || isAvailable != true {
		panic(err)
	}
	mountDir := yummyAgentConfig.AgentConfigMap["agentConfigMap"].MountDir
	stat, err := os.Stat(mountDir)
	if os.IsNotExist(err) || !stat.IsDir() {
		panic("mount dir not exist or not dir")
	}

	controller.StartController(client, nodeName, vg, mountDir)
}

func getNode(client *kubernetes.Clientset, name string) *v1.Node {
	node, err := client.CoreV1().Nodes().Get(name, metav1.GetOptions{})
	if err != nil {
		glog.Fatalf("Could not get node information: %v", err)
	}
	return node
}
