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

package reader

import (
	"context"
	"fmt"
	"github.com/coreos/etcd-operator/pkg/backup/util"
	cos "github.com/tencentyun/cos-go-sdk-v5"
	"io"
	"net/url"
)

// ensure cosReader satisfies reader interface.
var _ Reader = &cosReader{}

type cosReader struct {
	cos *cos.Client
}

// NewCOSReader creates a cos reader.
func NewCOSReader(cos *cos.Client) Reader {
	return &cosReader{cos}
}

// Open opens the file on path where path must be in the format "<cos-bucket-name>/<key>"
func (cosr *cosReader) Open(path string) (io.ReadCloser, error) {
	bk, key, err := util.ParseBucketAndKey(path)
	if err != nil {
		return nil, err
	}
	fmt.Printf("bucket name is %s,key is %s\n", bk, key)
	//opt := &cos.ObjectGetOptions{
	//	ResponseContentType: "text/html",
	//}
	u, err := url.Parse("https://" + bk)
	if err != nil {
		return nil, err
	}
	cosr.cos.BaseURL = &cos.BaseURL{BucketURL: u}
	resp, err := cosr.cos.Object.Get(context.Background(), key, nil)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}
