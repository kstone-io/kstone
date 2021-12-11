# Kstone Installation

[中文](README_CN.md)

## 1 Preparation

- Apply for a cluster from [TKE](https://cloud.tencent.com/product/tke) or install [minikube](https://minikube.sigs.k8s.io/docs/start/).
- Requirements：
  - Worker >= 4 vCPU 8 GB of Memory.
  - Can access the managed etcd.
  
## 2 Install minikube (optional)

#### Step 1：

- Install minikube (e.g. Linux amd64)
  - Requirements：version <= 1.20.x
```shell
VERSION=v1.20.0
curl -LO https://storage.googleapis.com/minikube/releases/${VERSION}/minikube-linux-amd64
sudo install minikube-linux-amd64 /usr/local/bin/minikube
```

#### Step 2：

- Alias `minikube kubectl`
```shell
alias kubectl="minikube kubectl --"
```

## 3 Deploy

### 3.1 Modify Helm Configuration

#### Step 1：

- Download Helm Repo:

``` shell
git clone git@github.com:tkestack/kstone.git
cd ./charts
```

- Modify Setting:

``` yaml
// kstone-charts/values.yaml

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

- Fill in the TOKEN of the cluster to deploy.

``` yaml
// kstone-charts/charts/dashboard-api/values.yaml

kube:
  token: $token
  target: kubernetes.default.svc.cluster.local:443
```

- Requirements：
  - $token is the access credential TOKEN of the TKE cluster to be deployed.
  - $token needs to have access to all resources in the cluster.

### 3.2 Install

``` shell
cd kstone-charts

kubectl create ns kstone

helm install kstone . -n kstone
```

### 3.3 Update

``` shell
cd kstone-charts

helm upgrade kstone . -n kstone
```

### 3.4 Uninstall

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