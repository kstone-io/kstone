#!/usr/bin/env bash

set -ex
kubectl create ns kstone
cd charts && helm install kstone . -n kstone -f values.test.yaml --set global.kstone.tag=$VERSION
echo "kstone component images:"
kubectl get deployment -n kstone -o yaml | grep image: | grep -v "f:image:"
