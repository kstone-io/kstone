/*
 * Tencent is pleased to support the open source community by making TKEStack
 * available.
 *
 * Copyright (C) 2012-2023 Tencent. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use
 * this file except in compliance with the License. You may obtain a copy of the
 * License at
 *
 * https://opensource.org/licenses/Apache-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 * WARRANTIES OF ANY KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations under the License.
 */

package authentication

import (
	"context"
	"fmt"
	"k8s.io/client-go/kubernetes"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"tkestack.io/kstone/pkg/controllers/util"
)

var (
	once     sync.Once
	instance *DefaultStore
)

// Store gets and updates users
type Store interface {
	// UserGet gets a user
	UserGet(username string) (*User, error)
	// UserAdd adds a user
	UserAdd(user User) error
	// UserDelete deletes a user
	UserDelete(username string) error
	// UserList lists users
	UserList() ([]*User, error)
	// UserChangePassword changes a password of a user
	UserChangePassword(username, password string) error
}

type User struct {
	Name           string
	HashedPassword string
	ExtraInfo      map[string]interface{}
}

type DefaultStore struct {
	kubeCli kubernetes.Interface
}

func GetDefaultStoreInstance() *DefaultStore {
	once.Do(func() {
		instance = &DefaultStore{
			kubeCli: util.NewSimpleClientBuilder("").ClientOrDie(),
		}
	})
	return instance
}

func (s *DefaultStore) UserGet(username string) (*User, error) {
	cm, err := s.kubeCli.CoreV1().ConfigMaps(DefaultKstoneNamespace).Get(context.TODO(), DefaultConfigMapName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	hashedPassword, ok := cm.Data[username]
	if !ok {
		return nil, fmt.Errorf("failed to get user %s, user not exists", username)
	}
	return &User{
		Name:           username,
		HashedPassword: hashedPassword,
	}, err
}

func (s *DefaultStore) UserAdd(user User) error {
	cm, err := s.kubeCli.CoreV1().ConfigMaps(DefaultKstoneNamespace).Get(context.TODO(), DefaultConfigMapName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	_, ok := cm.Data[user.Name]
	if ok {
		return fmt.Errorf("failed to add user %s, user already exists", user.Name)
	}
	cm.Data[user.Name] = user.HashedPassword
	if _, err := s.kubeCli.CoreV1().ConfigMaps(DefaultKstoneNamespace).Update(context.TODO(), cm, metav1.UpdateOptions{}); err != nil {
		return err
	}
	return nil
}

func (s *DefaultStore) UserDelete(username string) error {
	cm, err := s.kubeCli.CoreV1().ConfigMaps(DefaultKstoneNamespace).Get(context.TODO(), DefaultConfigMapName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	_, ok := cm.Data[username]
	if !ok {
		return fmt.Errorf("failed to delete user %s, user not exists", username)
	}
	if len(cm.Data) == 1 {
		return fmt.Errorf("failed to delete user %s, only one user remains", username)
	}
	delete(cm.Data, username)
	if _, err := s.kubeCli.CoreV1().ConfigMaps(DefaultKstoneNamespace).Update(context.TODO(), cm, metav1.UpdateOptions{}); err != nil {
		return err
	}
	return nil
}

func (s *DefaultStore) UserList() ([]*User, error) {
	var userList []*User
	cm, err := s.kubeCli.CoreV1().ConfigMaps(DefaultKstoneNamespace).Get(context.TODO(), DefaultConfigMapName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	for u, p := range cm.Data {
		userList = append(userList, &User{
			Name:           u,
			HashedPassword: p,
		})
	}
	return userList, nil
}

func (s *DefaultStore) UserChangePassword(username, password string) error {
	cm, err := s.kubeCli.CoreV1().ConfigMaps(DefaultKstoneNamespace).Get(context.TODO(), DefaultConfigMapName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	_, ok := cm.Data[username]
	if !ok {
		return fmt.Errorf("failed to change password for user %s, user not exists", username)
	}
	cm.Data[username] = password
	if _, err := s.kubeCli.CoreV1().ConfigMaps(DefaultKstoneNamespace).Update(context.TODO(), cm, metav1.UpdateOptions{}); err != nil {
		return err
	}
	return nil
}
