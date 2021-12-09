# 安装Kstone

[英文](./)

## 1 资源准备

- 申请 [TKE](https://cloud.tencent.com/product/tke) 集群。
- 环境要求：
  - Worker 4C8G以上配置。
  - 可访问待管理的目标etcd。

## 2 部署

### 2.1 修改Helm配置

#### 步骤一：

- 下载Helm Repo：

``` shell
git clone git@github.com:tkestack/kstone.git
cd ./charts
```

- 修改配置：

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

方式一 使用现有LB

参考上述配置，填入TKE集群同VPC下的$lb和$subnet。

方式二 不使用现有LB

删除以下配置：

- ingress.annotations.service.kubernetes.io/tke-existed-lbid
- kubernetes.io/ingress.existLbId
- kubernetes.io/ingress.subnetId

#### 步骤二：

- 填入运行集群的TOKEN。

``` yaml
// kstone-charts/charts/dashboard-api/values.yaml

kube:
  token: $token
  target: kubernetes.default.svc.cluster.local:443
```

- 要求：
  - $token为即将部署的TKE集群的访问凭证TOKEN。
  - $token需要具备访问集群范围所有资源的权限。

### 2.2 安装

``` shell
cd kstone-charts

kubectl create ns kstone

helm install kstone . -n kstone
```

### 2.3 更新

``` shell
cd kstone-charts

helm upgrade kstone . -n kstone
```

### 2.4 卸载

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