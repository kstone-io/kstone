package credential

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type metadataResponse struct {
	TmpSecretId  string
	TmpSecretKey string
	Token        string
	ExpiredTime  int64
	Code         string
}

type Credential struct {
	sync.Mutex
	expiredTime int64
	id          string
	key         string
	token       string
	role        string
}

func NewCredential(role string) *Credential {
	return &Credential{
		role: role,
	}
}

func (c *Credential) GetSecret() (string, string, string, error) {
	c.Lock()
	defer c.Unlock()

	if time.Now().Unix() > c.expiredTime {
		if err := c.refresh(); err != nil {
			return "", "", "", err
		}
	}
	return c.id, c.key, c.token, nil
}

func (c *Credential) refresh() error {
	res, err := http.Get(fmt.Sprintf("http://metadata.tencentyun.com/meta-data/cam/service-role-security-credentials/%s", c.role))
	if err != nil {
		return errors.Wrap(err, "http get failed")
	}

	if res.StatusCode != 200 {
		_ = res.Body.Close()
		res, err = http.Get(fmt.Sprintf("http://metadata.tencentyun.com/meta-data/cam/security-credentials/%s", c.role))
		if err != nil {
			return errors.Wrap(err, "http get failed")
		}
		if res.StatusCode != 200 {
			return fmt.Errorf("status code is %d", res.StatusCode)
		}
	}

	defer func() { _ = res.Body.Close() }()

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return errors.Wrapf(err, "read data failed")
	}

	metaData := &metadataResponse{}
	if err := json.Unmarshal(data, metaData); err != nil {
		return errors.Wrapf(err, "unmarshal failed")
	}

	if metaData.Code != "Success" {
		return fmt.Errorf("get Code is %s", metaData.Code)
	}

	c.id = metaData.TmpSecretId
	c.key = metaData.TmpSecretKey
	c.token = metaData.Token
	c.expiredTime = metaData.ExpiredTime
	return nil
}
