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
	cos "github.com/tencentyun/cos-go-sdk-v5"
	"io"
	"net/http"
	"net/url"
	"strings"
)

var _ Writer = &cosWriter{}

type cosWriter struct {
	cos *cos.Client
}

// NewCOSWriter creates a cos writer.
func NewCOSWriter(cos *cos.Client) Writer {
	return &cosWriter{cos}
}

// Write writes the backup file to the given cos path, "<cos-bucket-name>/<key>".
func (cosw *cosWriter) Write(ctx context.Context, path string, r io.Reader) (int64, error) {
	bk, key, err := util.ParseBucketAndKey(path)
	if err != nil {
		return 0, err
	}
	fmt.Printf("bucket name is %s,key is %s\n", bk, key)

	u, err := url.Parse("https://" + bk)
	if err != nil {
		return 0, err
	}
	cosw.cos.BaseURL = &cos.BaseURL{BucketURL: u}

	// create bucket if not exist
	rsp, err := cosw.cos.Bucket.Head(ctx)
	if rsp != nil && rsp.StatusCode == http.StatusNotFound {
		_, err := cosw.cos.Bucket.Put(ctx, &cos.BucketPutOptions{
			XCosACL: "private",
		})
		if err != nil {
			return 0, err
		}
	} else if err != nil {
		return 0, err
	}

	optcom := &cos.CompleteMultipartUploadOptions{}

	v, _, err := cosw.cos.Object.InitiateMultipartUpload(ctx, key, nil)
	if err != nil {
		logrus.Errorf("failed to init multipart upload, err: %v", err)
		return 0, err
	}
	uploadID := v.UploadID


	count := 1
	p := make([]byte, 1 << 30)
	for {
		//n, err := r.Read(p)
		n, err := io.ReadFull(r, p)
		if err != nil  && err != io.ErrUnexpectedEOF{
			if err == io.EOF {
				logrus.Infof("meet EOF, complete reading data")
				break
			}
			logrus.Errorf("failed to read data, err: %v", err)
			return 0, err
		}

		temp := strings.NewReader(string(p[:n]))

		resp, err := cosw.cos.Object.UploadPart(ctx, key, uploadID, count, temp, nil)
		if err != nil {
			logrus.Errorf("failed to upload part %d, err: %v", count, err)
			return 0, err
		}

		optcom.Parts = append(optcom.Parts, cos.Object{
			PartNumber: count, ETag: resp.Header.Get("ETag"),
		})

		logrus.Infof("complete uploading part %d", count)

		count++
	}

	_, _, err = cosw.cos.Object.CompleteMultipartUpload(ctx, key, uploadID, optcom)
	if err != nil {
		logrus.Errorf("failed to complete uploading multipart, uploadID: %s, err: %v", uploadID, err)
		return 0, err
	}

	logrus.Infof("complete uploading the backup file, bucket name is %s, key is %s", bk, key)

	resp, err := cosw.cos.Object.Get(ctx, key, nil)
	if err != nil {
		return 0, err
	}
	resp.Body.Close()
	return resp.ContentLength, nil
}

func (cosw *cosWriter) Delete(ctx context.Context, path string) error {
	bk, key, err := util.ParseBucketAndKey(path)
	if err != nil {
		return err
	}

	u, err := url.Parse("https://" + bk)
	if err != nil {
		return err
	}
	cosw.cos.BaseURL = &cos.BaseURL{BucketURL: u}

	_, err = cosw.cos.Object.Delete(ctx, key)

	return err
}

// List return the file paths which match the given cos path
func (cosw *cosWriter) List(ctx context.Context, basePath string) ([]string, error) {
	bk, key, err := util.ParseBucketAndKey(basePath)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse("https://" + bk)
	if err != nil {
		return nil, err
	}
	cosw.cos.BaseURL = &cos.BaseURL{BucketURL: u}

	objects, _, err := cosw.cos.Bucket.Get(ctx, &cos.BucketGetOptions{Prefix: key})

	if err != nil {
		return nil, err
	}

	objectKeys := []string{}
	for _, object := range objects.Contents {
		objectKeys = append(objectKeys, bk+"/"+object.Key)
	}
	return objectKeys, nil
}
