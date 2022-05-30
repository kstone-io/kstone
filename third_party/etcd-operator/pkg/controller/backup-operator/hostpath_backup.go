// Copyright 2017 The etcd-operator Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"

	api "github.com/coreos/etcd-operator/pkg/apis/etcd/v1beta2"
	"github.com/coreos/etcd-operator/pkg/backup"
	"github.com/coreos/etcd-operator/pkg/backup/writer"

	"k8s.io/client-go/kubernetes"
)
const etcdBackFilePrefix = "etcdbackup"
// handleHostPath saves etcd cluster's backup to specificed host path.
func handleHostPath(ctx context.Context, kubecli kubernetes.Interface, s *api.HostPathBackupSource,endpoints []string, clientTLSSecret, basicAuthSecret, namespace string, insecureSkipVerify bool, isPeriodic bool, maxBackup int) (bs *api.BackupStatus, err error) {
	hostPath := os.Getenv("HOST_PATH_NAME")
	if len(hostPath) == 0 {
		return nil, fmt.Errorf("HOST_PATH_NAME env must be set")
	}
	
	var tlsConfig *tls.Config
	if tlsConfig, err = generateTLSConfigWithVerify(kubecli, clientTLSSecret, namespace, insecureSkipVerify); err != nil {
		return nil, err
	}

	var username, password string
	if username, password, err = generateUsernamePassword(kubecli, basicAuthSecret, namespace); err != nil {
		return nil, err
	}

	bm := backup.NewBackupManagerFromWriter(kubecli, writer.NewHostPathWriter(hostPath), tlsConfig, endpoints, namespace, username, password)

	rev, etcdVersion, now, err := bm.SaveSnap(ctx, hostPath + etcdBackFilePrefix, true)
	if err != nil {
		return nil, fmt.Errorf("failed to save snapshot (%v)", err)
	}
	if maxBackup > 0 {
		//err := bm.EnsureMaxBackup(ctx, s.Path, maxBackup)
		err := bm.EnsureMaxBackup(ctx,hostPath, maxBackup)
		if err != nil {
			return nil, fmt.Errorf("succeeded in saving snapshot but failed to delete old snapshot (%v)", err)
		}
	}
	return &api.BackupStatus{EtcdVersion: etcdVersion, EtcdRevision: rev, LastSuccessDate: *now}, nil
}
