package backup

import (
	"encoding/json"
	"fmt"

	klog "k8s.io/klog/v2"
	kstonev1alpha2 "tkestack.io/kstone/pkg/apis/kstone/v1alpha2"
)

func GetBackupConfig(cluster *kstonev1alpha2.EtcdCluster) (*Config, error) {
	var err error
	cfg, found := cluster.Annotations[AnnoBackupConfig]
	if !found {
		err = fmt.Errorf(
			"backup config not found, annotation key %s not exists, namespace is %s, name is %s",
			AnnoBackupConfig,
			cluster.Namespace,
			cluster.Name,
		)
		klog.Errorf("failed to get backup config,cluster %s,err is %v", cluster.Name, err)
		return nil, err
	}

	backupConfig := &Config{}
	err = json.Unmarshal([]byte(cfg), backupConfig)
	if err != nil {
		klog.Errorf(err.Error())
		return nil, err
	}
	return backupConfig, nil
}
