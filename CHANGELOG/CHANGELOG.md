## v0.2.0-beta.2 (2022-06-08)
<hr>

### Features
<hr>

support import etcd v2 cluster. (@engow, @seanyan )

add compaction,lease metrics. (@tangcong )

add hostPath type for etcd backup operator. (@GeorgeGuo2018 )

### Bug Fix

failed to add/remove member if affinity and toleration is nil. (@engow )

fixed the issue that the editing page could not display cpu and memory resources. (@engow )

fixed the issue that some paths data could not be viewed visually. (@engow )

fixed the issue of backup management page 504 error.(@engow)



## v0.2.0-beta.1 (2022-04-18)
<hr>

### Breaking Changes
<hr>

add new API v1alpha2, refer to [migration document](../docs/migration/README.md) to migrate imported clusters API from v1alpha1 to v1alpha2. (@maudL1n, @lianghao208 )

### Features
<hr>

support kstone-api and kstone-dashboard authentication, default username: ***admin***, default password: ***adm1n@kstone.io***. (@engow, @lianghao208 )

support multiple namespaces. (@engow )

support auth and tolerations, affinity. (@engow )

support etcd cluster password authentication. (@lianghao208 )

add s3 backup and backup check. (@jianhaiqing )

### Test
<hr>

add e2e test for https kstone-etcd-operator cluster. (@maudL1n )
