// Copyright 2019 The etcd-operator Authors
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

package reader

import (
	"fmt"
	"io"

	"github.com/coreos/etcd-operator/pkg/backup/util"
)

// ensure hostPathReader satisfies reader interface.
var _ Reader = &hostPathReader{}

// hostPathReader provides Reader imlementation for reading a file from S3
type hostPathReader struct {
	basePath string
}

// NewHostPathReader return a Reader implementation to read a file from hostPath in the form of hostPathReader
func NewHostPathReader(path string) Reader {
	return &hostPathReader{path}
}

// Open opens the file on path where path must be in the format "/data/etcd.backup/xxxx"
func (hostPathr *hostPathReader) Open(path string) (io.ReadCloser, error) {
	file,err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read hostPath file %s: %v", path,err)
	}
	return ioutil.NopCloser(bytes.NewReader(file)),err
}
