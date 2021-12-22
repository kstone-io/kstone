# Kstone Installation

[中文](README_CN.md)

## 1 Preparation

- Prerequisites
  - Kubernetes version is between 1.14 and 1.20.
  - The version of Prometheus Operator is v0.49.0.
- Apply for a cluster from [TKE](https://cloud.tencent.com/product/tke) or install [Minikube](https://minikube.sigs.k8s.io/docs/start/).
  - Kstone supports deploy in various cloud vendors and bare k8s cluster environments
  - In the environments mentioned above, only the corresponding ingress rules need to be configured.
- Requirements：
  - For production environment (recommended): Worker >= 4 vCPU 8 GB of Memory.
  - For test environment (minimum): Worker >= 2 vCPU 2 GB of Memory.
  - Can access the managed etcd.
- Safety warning:
  - The current version of kstone-dashboard does not support authentication.
  Please pay attention to data security and try not to expose it to the public network.
  
## 2 Install on TKE

[Kstone installation on TKE](../docs/installation/tke.md)

## 3 Install on Minikube

[Kstone installation on Minikube(Mac OS X)](../docs/installation/minikube-macos.md)

[Kstone installation on Minikube(Linux amd64)](../docs/installation/minikube-amd64.md)

## 4 Install on the cluster created by kubeadm

[Kstone installation on the cluster created by kubeadm](../docs/installation/kubeadm_en.md)