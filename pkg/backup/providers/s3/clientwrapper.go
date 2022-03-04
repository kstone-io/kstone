package s3

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/aws/aws-sdk-go/aws/session"
	awsS3 "github.com/aws/aws-sdk-go/service/s3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	tmpdir = "/tmp"
)

// ClientS3Wrapper is a wrapper for S3 client that provides cleanup functionality.
type ClientS3Wrapper struct {
	S3        *awsS3.S3
	configDir string
}

// NewClientFromSecret returns a S3 client based on given k8s secret containing aws credentials.
func NewClientFromSecret(kubecli kubernetes.Interface, namespace, endpoint, awsSecret string, forcePathStyle bool) (w *ClientS3Wrapper, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("new S3 client failed: %v", err)
		}
	}()
	w = &ClientS3Wrapper{}
	w.configDir, err = ioutil.TempDir(tmpdir, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create aws config dir: (%v)", err)
	}
	so, err := setupAWSConfig(kubecli, namespace, awsSecret, endpoint, w.configDir, forcePathStyle)
	if err != nil {
		return nil, fmt.Errorf("failed to setup aws config: (%v)", err)
	}
	sess, err := session.NewSessionWithOptions(*so)
	if err != nil {
		return nil, fmt.Errorf("new AWS session failed: %v", err)
	}
	w.S3 = awsS3.New(sess)
	return w, nil
}

// Close cleans up all intermediate resources for creating S3 client.
func (w *ClientS3Wrapper) Close() {
	os.RemoveAll(w.configDir)
}

// setupAWSConfig setup local AWS config/credential files from Kubernetes aws secret.
func setupAWSConfig(kubecli kubernetes.Interface, ns, secret, endpoint, configDir string, forcePathStyle bool) (*session.Options, error) {
	options := &session.Options{}
	options.SharedConfigState = session.SharedConfigEnable

	// empty string defaults to aws
	options.Config.Endpoint = &endpoint

	options.Config.S3ForcePathStyle = &forcePathStyle

	se, err := kubecli.CoreV1().Secrets(ns).Get(context.TODO(), secret, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("setup AWS config failed: get k8s secret failed: %v", err)
	}

	creds := se.Data["credentials"]
	if len(creds) != 0 {
		credsFile := path.Join(configDir, "credentials")
		err = ioutil.WriteFile(credsFile, creds, 0600)
		if err != nil {
			return nil, fmt.Errorf("setup AWS config failed: write credentials file failed: %v", err)
		}
		options.SharedConfigFiles = append(options.SharedConfigFiles, credsFile)
	}

	config := se.Data["config"]
	if len(config) != 0 {
		configFile := path.Join(configDir, "config")
		err = ioutil.WriteFile(configFile, config, 0600)
		if err != nil {
			return nil, fmt.Errorf("setup AWS config failed: write config file failed: %v", err)
		}
		options.SharedConfigFiles = append(options.SharedConfigFiles, configFile)
	}

	return options, nil
}
