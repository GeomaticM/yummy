package lvm

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/nak3/go-lvm"
)

func CreateLv(vg string, lvName string, size int64) (*lvm.LvObject, error) {
	vgo := &lvm.VgObject{}

	glog.Infof("create lv from vg %s, name %s size %d", vg, lvName, size)

	// Open VG in write mode
	vgo.Vgt = lvm.VgOpen(vg, "w")
	defer vgo.Close()

	// Create a LV
	l, err := vgo.CreateLvLinear(lvName, size)
	if err != nil {
		glog.Infof("error: %v", err)
		return nil, err
	}

	glog.Infof("new lv uuid %s", l.GetUuid())
	return l, nil
}

func DeleteLv(vg string, lvName string) {
	vgo := &lvm.VgObject{}

	glog.Infof("delete lv %s from vg %s", lvName, vg)

	// Open VG in read mode
	vgo.Vgt = lvm.VgOpen(vg, "w")
	defer vgo.Close()

	lv, err := vgo.LvFromName(lvName)
	if err != nil {
		glog.Warningf("get lv %s failed, skip deleting", lvName)
		return
	}

	err = lv.Remove()
	if err != nil {
		glog.Warning("lv delete failed", err)
	}
	glog.Infof("lv delete success")
}

func IsLvExist(vg string, lvName string) bool {
	vgo := &lvm.VgObject{}

	glog.Infof("check lv %s exist in vg %s", lvName, vg)

	// Open VG in read mode
	vgo.Vgt = lvm.VgOpen(vg, "r")
	defer vgo.Close()

	lv, err := vgo.LvFromName(lvName)
	if err == nil && lv.IsActive() {
		glog.Info("lv exist and is active")
		return true
	}

	return false
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
	//todo: get vg by name
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

func GetVgInfo(vg string) (vgSize, vgFree uint64, err error) {
	//todo: get vg by name
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
				return vgSize, vgFree, nil
			} else {
				return 0, 0, fmt.Errorf("vg size incorrect, size %v free %v", vgSize, vgFree)
			}
		}
	}

	fmt.Printf("no VG that has free space found\n")
	return 0, 0, fmt.Errorf("VG %s is not avaliable", vg)
}
