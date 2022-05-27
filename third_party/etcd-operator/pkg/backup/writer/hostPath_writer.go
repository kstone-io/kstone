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

package writer

import (
	"context"
	"fmt"
	"github.com/coreos/etcd-operator/pkg/backup/util"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

var _ Writer = &hostPathWriter{}

type hostPathWriter struct {
	basePath string
}

// NewHostPathSWriter creates a hostPath writer.
func NewHostPathWriter(path string) Writer {
	return &hostPathWriter{path}
}

// Write writes the backup file to the given path, "/data/etcd.backup/<key>".
func (hostPathw *hostPathWriter) Write(ctx context.Context, path string, r io.Reader) (int64, error) {
	w,err := os.Create(path)
	if err != nil{
		err = fmt.Errorf("failed to create backup file of hostPath: %v", err)
		return 0, err
	}

	defer w.Close()
	n, err := io.Copy(w, r)
	if err != nil {
		err = fmt.Errorf("failed to write to hostPath: %v", err)
	}
	return n, err
}

func (cosw *cosWriter) Delete(ctx context.Context, path string) error {
	err := os.Remove(path)
    if err != nil {
		err = fmt.Errorf("failed to delete deprecated backup file of hostPath: %v", err)	
    }

	return err
}

// List return the file paths which match the given host path
func (cosw *cosWriter) List(ctx context.Context, basePath string) ([]string, error) {
	files, err := ioutil.ReadDir(basePath)
    if err != nil {
		return nil, fmt.Errorf("failed to get backup files under path:%s", basePath)
    }


	results := []string{}
	for _, file := range files {
		results = append(results, file)
	}
	return results, nil
}
