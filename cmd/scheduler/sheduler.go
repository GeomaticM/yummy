package main

import (
	"github.com/silenceshell/yummy/pkg/scheduler/controller"
	"github.com/silenceshell/yummy/pkg/utils"
	"time"
	"flag"
	"math/rand"
	"github.com/golang/glog"
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