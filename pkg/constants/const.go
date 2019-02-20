package constants

// yummy.vg.size=128.00 GiB
// yummy.pe.size=4.00 MiB
// yummy.alloc.pe=2730 / 10.66 GiB
// yummy.free.pe=30037 / 117.33 GiB

const (
	AnnotationKey = "yummyNodeName"
	AnnotationVgSize = "yummy.vg.size"
	AnnotationPeSize = "yummy.pe.size"  	//MiB
	AnnotationAllocPe = "yummy.alloc.pe"
	AnnotationFreePe = "yummy.free.pe"		//count
)
