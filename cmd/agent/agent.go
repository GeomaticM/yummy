package main

import "C"

import (
	"flag"
	"github.com/golang/glog"
	"github.com/silenceshell/yummy/pkg/constants"
	"github.com/silenceshell/yummy/pkg/utils"
	"strconv"

	//"github.com/prometheus/client_golang/prometheus"
	//"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/silenceshell/yummy/pkg/agent/common"
	"github.com/silenceshell/yummy/pkg/agent/controller"
	"github.com/silenceshell/yummy/pkg/agent/lvm"
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

	client := utils.SetupClient()
	node := utils.GetNode(client, nodeName)

	glog.Info("node name is ", node.Name)

	vg := yummyAgentConfig.AgentConfigMap["agentConfigMap"].VolumeGroup
	isAvailable, err := lvm.IsVgExistAndAvailable(vg)
	if err != nil || isAvailable != true {
		panic(err)
	}

	// todo:Get vg info and set node annotation such as:
	// yummy.vg.size=128.00 GiB
	// yummy.pe.size=4.00 MiB
	// yummy.alloc.pe=2730 / 10.66 GiB
	// yummy.free.pe=30037 / 117.33 GiB
	// this annotation will be updated every time when pvc is created

	vgSize, vgFree, err := lvm.GetVgInfo(vg)
	if err != nil {
		panic(err)
	}
	node.Annotations[constants.AnnotationVgSize] = strconv.FormatUint(vgSize, 10)
	node.Annotations[constants.AnnotationVgFreeSize] = strconv.FormatUint(vgFree, 10)

	_, err = client.CoreV1().Nodes().Update(node)
	if err != nil {
		panic(err)
	}

	mountDir := yummyAgentConfig.AgentConfigMap["agentConfigMap"].MountDir
	stat, err := os.Stat(mountDir)
	if os.IsNotExist(err) || !stat.IsDir() {
		panic("mount dir not exist or not dir")
	}

	//glog.Infof("Starting metrics server at %s\n", optListenAddress)
	//prometheus.MustRegister([]prometheus.Collector{
	//	metrics.PersistentVolumeDiscoveryTotal,
	//	metrics.PersistentVolumeDiscoveryDurationSeconds,
	//	metrics.PersistentVolumeDeleteTotal,
	//	metrics.PersistentVolumeDeleteDurationSeconds,
	//	metrics.PersistentVolumeDeleteFailedTotal,
	//	metrics.APIServerRequestsTotal,
	//	metrics.APIServerRequestsFailedTotal,
	//	metrics.APIServerRequestsDurationSeconds,
	//	collectors.NewProcTableCollector(procTable),
	//}...)
	//http.Handle(optMetricsPath, promhttp.Handler())

	controller.StartController(client, nodeName, vg, mountDir)
}
