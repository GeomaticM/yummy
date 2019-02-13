package utils

import (
	"github.com/golang/glog"
	"os/exec"
)

func Run(cmd string, args ...string) error {
	glog.Infof("Running %s %s", cmd, args)
	out, err := exec.Command(cmd, args...).CombinedOutput()
	if err != nil {
		glog.Errorf("Error running %s %v: %v, %s", cmd, args, err, out)
	}
	return err
}