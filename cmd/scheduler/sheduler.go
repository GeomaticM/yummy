package main

import (
	"flag"
	"github.com/golang/glog"
	"github.com/silenceshell/yummy/pkg/scheduler/controller"
	"github.com/silenceshell/yummy/pkg/utils"
	"math/rand"
	"time"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	flag.Set("logtostderr", "true")
	flag.Parse()

	glog.Infof("Ready to run...")

	client := utils.SetupClient()
	controller.StartPvcController(client)

	// listen node

	// listen pvc

	// pvc controller
	// calculate which node is fit for this pvc and set annotation on this pvc
}
