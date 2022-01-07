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
	"k8s.io/apimachinery/pkg/api/resource"

	kstonev1alpha1 "tkestack.io/kstone/pkg/apis/kstone/v1alpha1"
)

const (
	QosBurstable  = "Burstable"
	QosGuaranteed = "Guaranteed"
)

// CalculateCPURequests calculate cpuRequests
func CalculateCPURequests(totalCPU, ratio uint) string {
	return resource.NewMilliQuantity(int64(1000*totalCPU/ratio), resource.DecimalSI).String()
}

// CalculateMemoryRequests calculate memoryRequests
func CalculateMemoryRequests(totalMemory, ratio uint) string {
	return resource.NewQuantity(int64(1024*1024*1024*totalMemory/ratio), resource.BinarySI).String()
}

// GetQosRatio returns QosRatio
func GetQosRatio(cluster *kstonev1alpha1.EtcdCluster) uint {
	var ratio uint = 1
	if cluster.Spec.QosClass == QosBurstable {
		if cluster.Spec.QosRatio > 1 {
			ratio = cluster.Spec.QosRatio
		}
		if ratio > 100 {
			ratio = 100
		}
	}
	return ratio
}

// CheckResourceEqual parse resources and checks if it is equal
func CheckResourceEqual(actual, desired string) (bool, error) {
	actualQuantity, err := resource.ParseQuantity(actual)
	if err != nil {
		return false, err
	}
	desiredQuantity, err := resource.ParseQuantity(desired)
	if err != nil {
		return false, err
	}
	return actualQuantity.Equal(desiredQuantity), nil
}
