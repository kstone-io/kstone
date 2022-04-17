# API v1alpha2 Migration (Imported ETCD Clusters Only)

## 1 Preparation

- Prerequisites
    - Already installed the [release-0.1.0](https://github.com/tkestack/kstone/releases/tag/v0.1.0-alpha.2) or lower version kstone.
    - Already installed kubectl and set up the kubeconfig in /root/.kube/config for kstone cluster.
    
> **Risk Warning:**
> 
> Migrating API from v1alpha1 to v1alpha2 will delete the existing clusters from kstone.
> 
> Please notice this migration document is only for **Imported etcd** etcd clsuters migration.

## 2 Backup and Recreate the cluster and secret from v1alpha1 API

Run the migration script:

```shell
./hack/migratev1.sh
```

the outputs:

```shell
==========Starting to backup...============
==========Starting to delete clusters, inspections and secrets...==========
customresourcedefinition.apiextensions.k8s.io "etcdclusters.kstone.tkestack.io" deleted
customresourcedefinition.apiextensions.k8s.io "etcdinspections.kstone.tkestack.io" deleted
==========Starting to re-create crds...==========
customresourcedefinition.apiextensions.k8s.io/etcdclusters.kstone.tkestack.io created
==========Starting to generate clusters...==========
etcdcluster.kstone.tkestack.io/test-kstone created
==========Starting to generate secrets...==========
secret/xxx created
==========finished==========
```

## 3 Upgrade kstone by helm

- Helm upgrade for production environment

``` shell
cd charts

helm upgrade kstone . -n kstone -f values.yaml
```

or

- Helm upgrade for test environment

``` shell
cd charts

helm upgrade kstone . -n kstone -f values.test.yaml
```