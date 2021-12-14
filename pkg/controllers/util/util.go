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

package util

import (
	"fmt"
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/diff"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"
	fake "k8s.io/client-go/kubernetes/fake"
	restclient "k8s.io/client-go/rest"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	"tkestack.io/kstone/pkg/k8s"
)

const (
	ComponentEtcdClusterController    = "etcdcluster-controller"
	ComponentEtcdInspectionController = "etcdinspection-controller"
)

type EtcdClusterPhase string

const (
	EtcdClusterCreating     EtcdClusterPhase = "EtcdClusterCreating"
	EtcdClusterUpdating     EtcdClusterPhase = "EtcdClusterUpdating"
	EtcdClusterUpdateStatus EtcdClusterPhase = "EtcdClusterUpdateStatus"
)

const (
	ClusterTLSSecretName      = "certName"
	ClusterExtensionClientURL = "extClientURL"
)

type ClientBuilder interface {
	ConfigOrDie() *restclient.Config
	ClientOrDie() clientset.Interface
	DynamicClientOrDie() dynamic.Interface
}

func NewSimpleClientBuilder(kubeconfig string) ClientBuilder {
	builder := &simpleClientBuilder{
		kubeconfig: kubeconfig,
	}
	return builder
}

type simpleClientBuilder struct {
	kubeconfig string
}

func (s simpleClientBuilder) ConfigOrDie() *restclient.Config {
	cfg, err := k8s.GetClientConfig(s.kubeconfig)
	if err != nil {
		klog.Fatal(err)
	}
	return cfg
}

func (s simpleClientBuilder) ClientOrDie() clientset.Interface {
	clientConfig := s.ConfigOrDie()
	client, err := clientset.NewForConfig(clientConfig)
	if err != nil {
		klog.Fatal(err)
	}
	return client
}

func (s simpleClientBuilder) DynamicClientOrDie() dynamic.Interface {
	clientConfig := s.ConfigOrDie()
	client, err := dynamic.NewForConfig(clientConfig)
	if err != nil {
		klog.Fatal(err)
	}
	return client
}

func NewFakeClientBuilder() ClientBuilder {
	builder := &fakeClientBuilder{
		client: fake.NewSimpleClientset([]runtime.Object{}...),
	}
	return builder
}

type fakeClientBuilder struct {
	client *fake.Clientset
}

func (f fakeClientBuilder) ConfigOrDie() *restclient.Config {
	// TODO: implement it and add controllers unit test
	return nil
}

func (f fakeClientBuilder) ClientOrDie() clientset.Interface {
	return f.client
}

func (f fakeClientBuilder) DynamicClientOrDie() dynamic.Interface {
	// TODO: implement it and add controllers unit test
	return nil
}

// CheckAction verifies that expected and actual actions are equal and both have
// same attached resources
func CheckAction(expected, actual core.Action, t *testing.T) {
	if !(expected.Matches(actual.GetVerb(),
		actual.GetResource().Resource) && actual.GetSubresource() == expected.GetSubresource()) {
		t.Errorf("Expected\n\t%#v\ngot\n\t%#v", expected, actual)
		return
	}

	if reflect.TypeOf(actual) != reflect.TypeOf(expected) {
		t.Errorf("Action has wrong type. Expected: %t. Got: %t", expected, actual)
		return
	}

	switch a := actual.(type) {
	case core.CreateActionImpl:
		e, _ := expected.(core.CreateActionImpl)
		expObject := e.GetObject()
		object := a.GetObject()

		if !reflect.DeepEqual(expObject, object) {
			t.Errorf("Action %s %s has wrong object\nDiff:\n %s",
				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintSideBySide(expObject, object))
		}
	case core.UpdateActionImpl:
		e, _ := expected.(core.UpdateActionImpl)
		expObject := e.GetObject()
		object := a.GetObject()

		if !reflect.DeepEqual(expObject, object) {
			t.Errorf("Action %s %s has wrong object\nDiff:\n %s",
				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintSideBySide(expObject, object))
		}
	case core.PatchActionImpl:
		e, _ := expected.(core.PatchActionImpl)
		expPatch := e.GetPatch()
		patch := a.GetPatch()

		if !reflect.DeepEqual(expPatch, patch) {
			t.Errorf("Action %s %s has wrong patch\nDiff:\n %s",
				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintSideBySide(expPatch, patch))
		}
	default:
		t.Errorf("Uncaptured Action %s %s, you should explicitly add a case to capture it",
			actual.GetVerb(), actual.GetResource().Resource)
	}
}

// ProcessWorkQueue handle item in queue, handle the result of syncHandler
func ProcessWorkQueue(
	queue workqueue.RateLimitingInterface,
	syncHandler func(eKey string) error,
	obj interface{}) error {
	defer queue.Done(obj)
	var key string
	var ok bool

	if key, ok = obj.(string); !ok {
		queue.Forget(obj)
		utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
		return nil
	}
	if err := syncHandler(key); err != nil {
		queue.AddRateLimited(key)
		return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
	}
	queue.Forget(obj)
	klog.V(4).Infof("Successfully synced '%s'", key)
	return nil
}

// FilterInformerActions filters list and watch actions for testing resources.
// Since list and watch don't change resource state we can filter it to lower
// nose level in our tests.
func FilterInformerActions(actions []core.Action, resourceName string) []core.Action {
	ret := []core.Action{}
	for _, action := range actions {
		if len(action.GetNamespace()) == 0 &&
			(action.Matches("list", resourceName) ||
				action.Matches("watch", resourceName)) {
			continue
		}
		ret = append(ret, action)
	}

	return ret
}

func CompareActions(t *testing.T, actions []core.Action, filteredActions []core.Action) {
	for i, action := range filteredActions {
		t.Logf("client action id %d,action info %+v", i, action)
		if len(actions) < i+1 {
			t.Errorf("%d unexpected filteredActions: %+v", len(filteredActions)-len(actions), filteredActions[i:])
			break
		}

		expectedAction := actions[i]
		CheckAction(expectedAction, action, t)
	}

	if len(actions) > len(filteredActions) {
		t.Errorf(
			"%d additional expected filteredActions:%+v",
			len(actions)-len(filteredActions),
			actions[len(filteredActions):],
		)
	}
}
