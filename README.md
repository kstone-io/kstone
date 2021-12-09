# Kstone

<div align=center><img width=800 hight=300 src="docs/images/icon.png" /></div>

------

[中文](README_CN.md)

Kstone is an [etcd](https://github.com/etcd-io/etcd) management platform, providing cluster management, monitoring, backup, inspection, data migration, visual viewing of etcd data, and intelligent diagnosis.

Kstone will help you efficiently manage etcd clusters, significantly reduce operation and maintenance costs, discover potential hazards in time, and improve the stability and user experience of k8s etcd storage.

------

## Features

* Supports registration of existing clusters and creation of new etcd clusters.
* Support prometheus monitoring, built-in rich etcd grafana panel diagram.
* Support multiple data backup methods (minute-level backup to object storage, real-time backup by deploying learner).
* Support multiple inspection strategies (data consistency, health, hot write requests, number of resource objects, etc.).
* Built-in web console and visual view etcd data.
* Lightweight, easy to install.
* Support data migration(to do list).
* Support intelligent diagnosis(to do list).


## Architecture

Kstone consists of 5 components: kstone-etcdcluster-controller,kstone-etcd-operator,kstone-etcdinspection-controller,kstone-api, kstone-dashboard.

![Architecture Of Kstone](docs/images/kstone-arch.png)

## Components

### [kstone](https://github.com/tkestack/kstone)

kstone consists of kstone-etcdcluster-controller,kstone-etcdinspection-controller,kstone-api.

### [kstone-etcd-operator](https://github.com/tkestack/kstone-etcd-operator)

kstone-etcd-operator provides rich etcd cluster management capabilities.It will also be open source soon.

### [kstone-dashboard](https://github.com/tkestack/kstone-dashboard)

The web management system provided by kstone is as follows:

![kstone-ui](docs/images/kstone-ui.png)

## Installation

Please read [the detailed installation document](charts),
You can quickly install kstone through helm.

## Developing

### Build

``` shell
mkdir -p ~/tkestack
cd ~/tkestack
git clone https://github.com/tkestack/kstone
cd kstone
make
```

## Community

* You are encouraged to communicate most things via GitHub [issues](https://github.com/tkestack/kstone/issues/new/choose) or [pull requests](https://github.com/tkestack/kstone/pulls).

## Licensing

Kstone is licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for the full license text.

