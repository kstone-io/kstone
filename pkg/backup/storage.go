package backup

import (
	"k8s.io/client-go/kubernetes"

	"tkestack.io/kstone/pkg/apis/kstone/v1alpha2"
)

// Storage is an abstract, pluggable interface for etcd backup storage.
type Storage interface {
	// List gets all backup files from object storage.
	List(cluster *v1alpha2.EtcdCluster) (interface{}, error)

	// Stat counts the number of backup file in the last day.
	Stat(objects interface{}) (int, error)
}

type StorageConfig struct {
	KubeCli kubernetes.Interface
}
