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

package v1alpha2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EtcdCluster is the Schema for the etcdclusters API
type EtcdCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   EtcdClusterSpec   `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status EtcdClusterStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

type EtcdClusterPhase string

const (
	EtcdClusterInit      EtcdClusterPhase = "Initing"
	EtcdCluterCreating   EtcdClusterPhase = "Creating"
	EtcdClusterRunning   EtcdClusterPhase = "Running"
	EtcdClusterUpdating  EtcdClusterPhase = "Updating"
	EtcdClusterDeleteing EtcdClusterPhase = "Deleting"
	EtcdClusterDeleted   EtcdClusterPhase = "Deleted"
	EtcdClusterUnknown   EtcdClusterPhase = "Unknown"   // connection refused or other errors
	EtcdClusterUnhealthy EtcdClusterPhase = "UnHealthy" // node health check returns unhealthy
)

type EtcdClusterConditionType string

const (
	EtcdClusterConditionCreate EtcdClusterConditionType = "Create"
	EtcdClusterConditionImport EtcdClusterConditionType = "Import"
	EtcdClusterConditionUpdate EtcdClusterConditionType = "Update"
	EtcdClusterConditionDelete EtcdClusterConditionType = "Delete"
)

// EtcdClusterCondition contains condition information for a EtcdCluster.
type EtcdClusterCondition struct {
	// Type of EtcdCluster condition.
	Type EtcdClusterConditionType `json:"type" protobuf:"bytes,1,opt,name=type,casttype=EtcdClusterPhase"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status" protobuf:"bytes,2,opt,name=status,casttype=k8s.io/api/core/v1.ConditionStatus"`
	// Last time we got an update on a given condition.
	// +optional
	StartTime metav1.Time `json:"startTime,omitempty" protobuf:"bytes,3,opt,name=startTime"`
	// Last time the condition transit from one status to another.
	// +optional
	EndTime metav1.Time `json:"endTime,omitempty" protobuf:"bytes,4,opt,name=endTime"`
	// (brief) reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty" protobuf:"bytes,5,opt,name=reason"`
	// Human readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,6,opt,name=message"`
}

type EtcdClusterType string

const (
	EtcdClusterKstone   EtcdClusterType = "kstone-etcd-operator"
	EtcdClusterImported EtcdClusterType = "imported"
)

// EtcdClusterSpec defines the desired state of EtcdCluster
type EtcdClusterSpec struct {
	Name        string `json:"name" protobuf:"bytes,1,opt,name=name"`               // etcd cluster name，uniqueKey
	Description string `json:"description" protobuf:"bytes,2,opt,name=description"` // etcd description

	AuthConfig AuthConfig `json:"authConfig,omitempty" protobuf:"bytes,3,opt,name=authConfig"` // tls config

	DiskType string `json:"diskType" protobuf:"bytes,4,opt,name=diskType"`  // disk type, CLOUD_SSD/CLOUD_PREMIUM/CLOUD_BASIC
	DiskSize uint   `json:"diskSize" protobuf:"varint,5,opt,name=diskSize"` // single node's disk size, unit: GB
	Size     uint   `json:"size"  protobuf:"varint,6,opt,name=size"`        // etcd cluster member count: support 1, 3, 5, 7

	Affinity   corev1.Affinity `json:"affinity,omitempty" protobuf:"bytes,7,opt,name=affinity"`
	Args       []string        `json:"args,omitempty" protobuf:"bytes,8,rep,name=args"`
	Env        []corev1.EnvVar `json:"env,omitempty" protobuf:"bytes,9,rep,name=env"`                // etcd environment variables
	Version    string          `json:"version" protobuf:"bytes,10,opt,name=version"`                 // etcd version
	Repository string          `json:"repository,omitempty" protobuf:"bytes,11,opt,name=repository"` // etcd image

	ClusterType EtcdClusterType `json:"clusterType" protobuf:"bytes,12,opt,name=clusterType,casttype=EtcdClusterType"` // ClusterType specifies the etcd cluster provider.

	Resources corev1.ResourceRequirements `json:"resources,omitempty" protobuf:"bytes,13,opt,name=resources"` // Resources specifies requests and limits
	// If specified, the pod's tolerations.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

// AuthConfig defines tls
type AuthConfig struct {
	EnableTLS bool     `json:"enableTLS,omitempty" protobuf:"varint,1,opt,name=enableTLS"`
	SAN       []string `json:"san,omitempty" protobuf:"bytes,2,rep,name=san"`
	TLSSecret string   `json:"tlsSecret,omitempty" protobuf:"bytes,3,opt,name=tlsSecret"`
}

type KStoneFeature string

const (
	KStoneFeatureAnno                      = "featureGates"
	KStoneFeatureMonitor     KStoneFeature = "monitor"
	KStoneFeatureBackup      KStoneFeature = "backup"
	KStoneFeatureHealthy     KStoneFeature = "healthy"
	KStoneFeatureConsistency KStoneFeature = "consistency"
	KStoneFeatureRequest     KStoneFeature = "request"
	KStoneFeatureAlarm       KStoneFeature = "alarm"
	KStoneFeatureBackupCheck KStoneFeature = "backupcheck"
)

// EtcdClusterStatus defines the actual state of EtcdCluster.
type EtcdClusterStatus struct {
	Conditions         []EtcdClusterCondition   `json:"conditions,omitempty" protobuf:"bytes,1,rep,name=conditions"`
	Phase              EtcdClusterPhase         `json:"phase" protobuf:"bytes,2,opt,name=phase,casttype=EtcdClusterPhase"`
	Members            []MemberStatus           `json:"members,omitempty" protobuf:"bytes,3,rep,name=members"`
	FeatureGatesStatus map[KStoneFeature]string `json:"featureGatesStatus,omitempty" protobuf:"bytes,4,rep,name=featureGatesStatus,castkey=KStoneFeature"`
	ServiceName        string                   `json:"serviceName,omitempty" protobuf:"bytes,5,opt,name=serviceName"`
}

type MemberPhase string

const (
	MemberPhaseUnStarted MemberPhase = "UnStarted" // etcd member list return unstarted
	MemberPhaseUnKnown   MemberPhase = "UnKnown"   // endpoint can not be connected
	MemberPhaseRunning   MemberPhase = "Running"   // health check returns healthy
	MemberPhaseUnHealthy MemberPhase = "UnHealthy" // health check returns unhealthy
)

type EtcdMemberRole string

const (
	EtcdMemberLearner EtcdMemberRole = "Learner"
	// kstone can get the list of etcd members, but cannot connect to the endpoint.
	EtcdMemberUnKnown  EtcdMemberRole = "UnKnown"
	EtcdMemberFollower EtcdMemberRole = "Follower"
	EtcdMemberLeader   EtcdMemberRole = "Leader"
)

type MemberStatus struct {
	Name               string         `json:"name" protobuf:"bytes,1,opt,name=name"`
	MemberId           string         `json:"memberId" protobuf:"bytes,2,opt,name=memberId"`
	Status             MemberPhase    `json:"status" protobuf:"bytes,3,opt,name=status,casttype=MemberPhase"`
	Version            string         `json:"version" protobuf:"bytes,4,opt,name=version"`
	Endpoint           string         `json:"endpoint" protobuf:"bytes,5,opt,name=endpoint"`
	Port               string         `json:"port" protobuf:"bytes,6,opt,name=port"`
	ClientUrl          string         `json:"clientUrl" protobuf:"bytes,7,opt,name=clientUrl"`
	ExtensionClientUrl string         `json:"extensionClientUrl" protobuf:"bytes,8,opt,name=extensionClientUrl"`
	Role               EtcdMemberRole `json:"role" protobuf:"bytes,9,opt,name=role,casttype=EtcdMemberRole"`
	Errors             []string       `json:"errors,omitempty" protobuf:"bytes,10,rep,name=errors"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EtcdClusterList contains a list of EtcdCluster
type EtcdClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []EtcdCluster `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:defaulter-gen=TypeMeta

// EtcdInspection is a specification for a EtcdInspection resource
type EtcdInspection struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   EtcdInspectionSpec   `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status EtcdInspectionStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// EtcdInspectionSpec is the spec for a EtcdInspectionSpec resource
type EtcdInspectionSpec struct {
	ClusterName        string `json:"clusterName" protobuf:"bytes,1,opt,name=clusterName"`
	InspectionType     string `json:"inspectionType" protobuf:"bytes,2,opt,name=inspectionType"`
	InspectionProvider string `json:"inspectionProvider,omitempty" protobuf:"bytes,3,opt,name=inspectionProvider"`
	IntervalInSecond   int    `json:"intervalInSecond,omitempty" protobuf:"varint,4,opt,name=intervalInSecond"`
}

// EtcdInspectionRecord is the status record for a EtcdInspectionRecord resource
type EtcdInspectionRecord struct {
	// Last time we got an update on a given condition.
	// +optional
	StartTime metav1.Time `json:"startTime,omitempty" protobuf:"bytes,1,opt,name=startTime"`
	// Last time the condition transit from one status to another.
	// +optional
	EndTime metav1.Time `json:"endTime,omitempty" protobuf:"bytes,2,opt,name=endTime"`
	// (brief) reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty" protobuf:"bytes,3,opt,name=reason"`
	// Human readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,4,opt,name=message"`
}

// EtcdInspectionStatus is the status for a EtcdInspectionStatus resource
type EtcdInspectionStatus struct {
	Reason          string                 `json:"reason,omitempty" protobuf:"bytes,1,opt,name=reason"`
	Message         string                 `json:"message,omitempty" protobuf:"bytes,2,opt,name=message"`
	Records         []EtcdInspectionRecord `json:"records,omitempty" protobuf:"bytes,3,rep,name=records"`
	LastUpdatedTime metav1.Time            `json:"lastUpdatedTime,omitempty" protobuf:"bytes,4,opt,name=lastUpdatedTime"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EtcdInspectionList is a list of EtcdInspection resources
type EtcdInspectionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	Items []EtcdInspection `json:"items" protobuf:"bytes,2,rep,name=items"`
}
