#!/bin/bash
DATE=`date +%Y%m%d%H%M%S`

mkdir -p ./migrate-${DATE} && cd ./migrate-${DATE}

echo "==========Starting to backup...============"

kubectl get cluster -n kstone -oyaml > cluster-v1-backup-${DATE}.yaml

sed 's#apiVersion: kstone.tkestack.io/v1alpha1#apiVersion: kstone.tkestack.io/v1alpha2#g' cluster-v1-backup-${DATE}.yaml > cluster-v2-backup-${DATE}.yaml

for i in `kubectl get cluster -n kstone --no-headers | awk '{print $1}'`;do kubectl get secret ${i} -n kstone -oyaml > secret-backup-${i}-${DATE}.yaml;done

for i in `ls secret-backup-*`;do sed '/ ownerReferences:/,/ uid:/d' ${i} > new-${i} ;done


echo "==========Starting to delete clusters, inspections and secrets...=========="

kubectl delete crd etcdclusters.kstone.tkestack.io etcdinspections.kstone.tkestack.io

echo "==========Starting to re-create crds...=========="

for i in `ls ../../deploy/crds/kstone.tkestack.io_*`;do kubectl create -f $i ;done

echo "==========Starting to generate clusters...=========="

kubectl create -f cluster-v2-backup-${DATE}.yaml

echo "==========Starting to generate secrets...=========="

for i in `ls new-secret-backup-*`;do kubectl create -f ${i} ;done

echo "==========finished=========="