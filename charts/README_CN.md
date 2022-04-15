# 安装Kstone

[英文](./)

## 1 资源准备

- 前置条件
  - Kubernetes集群版本在1.14和1.20之间。
  - Prometheus-Operator版本为v0.49.0。
- 申请 [TKE](https://cloud.tencent.com/product/tke) 集群或搭建 [minikube](https://minikube.sigs.k8s.io/docs/start/) 集群。
  - Kstone 支持部署在多云或原生 K8s 集群中
  - 只需要安装并配置相对应的 Ingress 转发规则即可
- 环境要求：
  - 生产环境配置要求（推荐）：Worker 4C8G以上配置。
  - 体验环境配置要求（最低）：Worker 2C2G以上配置。
  - 可访问待管理的目标etcd。

## 2 在 TKE 集群安装 Kstone

[Kstone installation on TKE](../docs/installation/tke.md)

## 3 在 Minikube 集群安装 Kstone

[Kstone installation on Minikube(Mac OS X)](../docs/installation/minikube-macos.md)

[Kstone installation on Minikube(Linux amd64)](../docs/installation/minikube-amd64.md)

## 4 在 kubeadm 集群安装 Kstone

[Kstone installation on the cluster created by kubeadm](../docs/installation/kubeadm_en.md)
