#!/usr/bin/env bash

set -ex
kubectl create ns kstone
cd charts && helm install kstone . -n kstone -f values.test.yaml --set global.kstone.tag=$VERSION
