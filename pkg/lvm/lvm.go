package lvm

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/nak3/go-lvm"
)

func CreateLv(vg string, lvName string, size int64) error {
	vgo := &lvm.VgObject{}

	glog.Infof("create lv from vg %s, name %s size %d", vg, lvName, size)

	// Open VG in write mode
	vgo.Vgt = lvm.VgOpen(vg, "w")
	defer vgo.Close()

	// Create a LV
	l, err := vgo.CreateLvLinear(lvName, size)
	if err != nil {
		glog.Infof("error: %v", err)
		return err
	}

	glog.Infof("new lv uuid %s", l.GetUuid())
	return nil
}

func DeleteLv() {
//lvremove /dev/volume-group1/default-myclaim2

}

func IsLvExist(vg string, lvName string) (bool, error) {
	vgo := &lvm.VgObject{}

	glog.Infof("check lv exist %s vg %s", vg, lvName)

	// Open VG in write mode
	//todo: consider to open vg once when agent start
	vgo.Vgt = lvm.VgOpen(vg, "r")
	defer vgo.Close()

	//lv, err := vgo.LvFromName(lvName)
	//if err != nil {
	//	glog.Warning("get lv failed", err)
	//	return false, err
	//}
	//if !lv.IsActive() {
	//	glog.Warning("lv not active")
	//	return false, nil
	//}
	//
	//glog.Infof("lv is exist and active %s", lv.GetUuid())

	lvs := vgo.ListLVs()
	for _, lv := range lvs {
		if lv == lvName {
			return true, nil
		}
	}

	return false, nil
}

func ListVgNames() {
	vglist := lvm.ListVgNames()
	availableVG := ""

	// Create a VG object
	vgo := &lvm.VgObject{}
	for i := 0; i < len(vglist); i++ {
		vgo.Vgt = lvm.VgOpen(vglist[i], "r")
		if vgo.GetFreeSize() > 0 {
			availableVG = vglist[i]
			vgo.Close()
			break
		}
		vgo.Close()
	}
	if availableVG == "" {
		fmt.Printf("no VG that has free space found\n")
		return
	}
	fmt.Println(availableVG)
}

var vgSize uint64
var vgFree uint64

func IsVgExistAndAvailable(vg string) (bool, error) {
	vglist := lvm.ListVgNames()

	// Create a VG object
	vgo := &lvm.VgObject{}
	for i := 0; i < len(vglist); i++ {
		if vglist[i] == vg {
			vgo.Vgt = lvm.VgOpen(vglist[i], "r")
			vgSize = uint64(vgo.GetSize())
			vgFree = uint64(vgo.GetFreeSize())
			vgo.Close()
			if vgFree > 0 && vgSize > 0 {
				fmt.Printf("vg size %v and free %v\r\n", vgSize, vgFree)
				return true, nil
			} else {
				return false, fmt.Errorf("vg size incorrect, size %v free %v", vgSize, vgFree)
			}
		}
	}

	fmt.Printf("no VG that has free space found\n")
	return false, fmt.Errorf("VG %s is not avaliable", vg)

}
