

	/*
	  --- Volume group ---
	  VG Name               volume-group1
	  VG Size               <128.00 GiB
	  PE Size               4.00 MiB
	  Total PE              32767
	  Alloc PE / Size       1382 / <5.40 GiB
	  Free  PE / Size       31385 / <122.60 GiB
	*/

pull or push?

when to create lvm?

master:
watch api server for pvc of all namespaces
then master will add annotation for pvc: yummyNodeName={node name}

yummyNodeName: ubuntu-2

agent filters those pvc for itself and create lv on the node
external storage local-volume-provisioner will find the new directory and create pv

update vg status every 30s


