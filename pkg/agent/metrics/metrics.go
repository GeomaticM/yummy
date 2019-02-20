package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// LocalVolumeProvisionerSubsystem is prometheus subsystem name.
	LocalVolumeProvisionerSubsystem = "local_volume_provisioner"
	// APIServerRequestCreate represents metrics related to create resource request.
	APIServerRequestCreate = "create"
	// APIServerRequestDelete represents metrics related to delete resource request.
	APIServerRequestDelete = "delete"
	// DeleteTypeProcess represents metrics releated deletion in process.
	DeleteTypeProcess = "process"
	// DeleteTypeJob represents metrics releated deletion by job.
	DeleteTypeJob = "job"
)

var (
	// PersistentVolumeDeleteDurationSeconds is used to collect latency in seconds to delete persistent volumes.
	PersistentVolumeDeleteDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: LocalVolumeProvisionerSubsystem,
			Name:      "persistentvolume_delete_duration_seconds",
			Help:      "Latency in seconds to delete persistent volumes. Broken down by persistent volume mode, delete type (process or job), capacity and cleanup_command.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"mode", "type", "capacity", "cleanup_command"},
	)
)