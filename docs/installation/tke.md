# Kstone Installation

[中文](README_CN.md)

## 1 Preparation

- Prerequisites
  - Kubernetes version is between 1.14 and 1.20.
  - The version of Prometheus Operator is v0.49.0.
- Apply for a cluster from [TKE](https://cloud.tencent.com/product/tke).
- Requirements：
  - For production environment (recommended): Worker >= 4 vCPU 8 GB of Memory.
  - For test environment (minimum): Worker >= 2 vCPU 2 GB of Memory.
  - Can access the managed etcd.

## 2 Deploy

### 2.1 Modify Helm Configuration

#### Step 1：
- Install helm:
  
Please refer to [helm installation](https://helm.sh/docs/intro/install/)

- Download Helm Repo:

``` shell
git clone -b release-0.1 git@github.com:tkestack/kstone.git
cd ./charts
```

- Modify Setting:

``` yaml
// charts/values.yaml

ingress:
  enabled: true
  className: ""
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /$2
    service.cloud.tencent.com/direct-access: 'false'
    kubernetes.io/ingress.class: qcloud
    service.kubernetes.io/tke-existed-lbid: $lb
    kubernetes.io/ingress.existLbId: $lb
    kubernetes.io/ingress.subnetId: $subnet
```

Method 1: Use existing LB

Refer to the above configuration and fill in the $lb and $subnet under the same VPC of the TKE cluster.

Method 2: Do not use existing LB

Delete the following configurations:

- ingress.annotations.service.kubernetes.io/tke-existed-lbid
- kubernetes.io/ingress.existLbId
- kubernetes.io/ingress.subnetId

#### Step 2：

- Get Admin TOKEN from TKE cluster

```shell
kubectl get secrets -o jsonpath="{.items[?(@.metadata.annotations['kubernetes\.io/service-account\.name']=='kube-admin')].data.token}" -n kube-system|base64 --decode
```
- Fill in the TOKEN of the cluster to deploy.

``` yaml
// charts/charts/dashboard-api/values.yaml

kube:
  token: $token
  target: kubernetes.default.svc.cluster.local:443
```

- Requirements：
    - $token is the access credential TOKEN of the TKE cluster to be deployed.
    - $token needs to have access to all resources in the cluster.

#### Step 3: Using the existing Prometheus Operator (optional)

- Set `prometheusOperator.enabled=false` in the file `charts/charts/kube-prometheus-stack/values.yaml`.
- Set `prometheus.enabled=false` in the file `charts/charts/kube-prometheus-stack/values.yaml`.
- Modify the file: `charts/charts/grafana/templates/configmap.yaml`, replace `http://{{ .Release.Name }}-prometheus-prometheus.{{ .Release.Namespace }}.svc.cluster.local:9090` to the query URL from the existing Prometheus Operator.

### 2.2 Install

- Create kstone namespace

``` shell
kubectl create ns kstone
```
- Helm install for production environment

```shell
cd charts/
helm install kstone . -n kstone -f values.yaml
```

or

- Helm install for test environment

```shell
cd charts/
helm install kstone . -n kstone -f values.test.yaml
```

### 2.3 Update

- Helm upgrade for production environment

```bash
cd charts

helm upgrade kstone . -n kstone -f values.yaml
```

or

- Helm upgrade for test environment

```bash
cd charts

helm upgrade kstone . -n kstone -f values.test.yaml
```

### 2.4 Uninstall

``` shell
helm uninstall kstone -n kstone

kubectl delete crd alertmanagerconfigs.monitoring.coreos.com
kubectl delete crd alertmanagers.monitoring.coreos.com
kubectl delete crd podmonitors.monitoring.coreos.com
kubectl delete crd probes.monitoring.coreos.com
kubectl delete crd prometheuses.monitoring.coreos.com
kubectl delete crd prometheusrules.monitoring.coreos.com
kubectl delete crd servicemonitors.monitoring.coreos.com
kubectl delete crd thanosrulers.monitoring.coreos.com
kubectl delete crd etcdclusters.kstone.tkestack.io
kubectl delete crd etcdinspections.kstone.tkestack.io
```