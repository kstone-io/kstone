#!/usr/bin/env bash

#
# Tencent is pleased to support the open source community by making TKEStack
# available.
#
# Copyright (C) 2012-2023 Tencent. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License"); you may not use
# this file except in compliance with the License. You may obtain a copy of the
# License at
#
# https://opensource.org/licenses/Apache-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
# WARRANTIES OF ANY KIND, either express or implied.  See the License for the
# specific language governing permissions and limitations under the License.
#

set -o errexit
set -o nounset
set -o pipefail

# generate-groups generates everything for a project with external types only, e.g. a project based
# on CustomResourceDefinitions.

if [[ "$#" -lt 4 ]] || [[ "${1}" == "--help" ]]; then
  cat <<EOF
Usage: $(basename "$0") <generators> <output-package> <internal-apis-package> <extensiona-apis-package> <groups-versions> ...
  <generators>        the generators comma separated to run (e.g. deepcopy-external,defaulter-external,client-external,
                      lister-external,informer-external,deepcopy-internal,defaulter-internal,client-internal,
                      lister-internal,informer-internal or all-external,all-internal,all).
  <output-package>    the output package name (e.g. github.com/example/project/pkg/generated).
  <int-apis-package>  the internal types dir (e.g. github.com/example/project/pkg/apis).
  <ext-apis-package>  the external types dir (e.g. github.com/example/project/pkg/apis or githubcom/example/apis).
  <groups-versions>   the groups and their versions in the format "groupA:v1,v2 groupB:v1 groupC:v2", relative
                      to <api-package>.
  ...                 arbitrary flags passed to all generator binaries.
Examples:
  $(basename "$0") all-external github.com/example/project/pkg/client github.com/example/project/pkg/apis github.com/example/project/apis "foo:v1 bar:v1alpha1,v1beta1"
  $(basename "$0") deepcopy-external,client-external github.com/example/project/pkg/client github.com/example/project/pkg/apis github.com/example/project/apis "foo:v1 bar:v1alpha1,v1beta1"
  $(basename "$0") all-internal github.com/example/project/pkg/client github.com/example/project/pkg/apis github.com/example/project/apis "foo:v1 bar:v1alpha1,v1beta1"
  $(basename "$0") deepcopy-internal,defaulter-internal,conversion-internal github.com/example/project/pkg/client github.com/example/project/pkg/apis github.com/example/project/apis "foo:v1 bar:v1alpha1,v1beta1"
EOF
  exit 0
fi

GENS="$1"
OUTPUT_PKG="$2"
INT_APIS_PKG="$3"
EXT_APIS_PKG="$4"
GROUPS_WITH_VERSIONS="$5"
shift 5

GOPATH=${GOPATH:-/go}
K8S_ROOT=${K8S_ROOT:-/go/src/k8s.io/kubernetes}
K8S_BIN=${K8S_ROOT}/_output/bin
PATH=${K8S_BIN}:${PATH}

function codegen_join() { local IFS="$1"; shift; echo "$*"; }

# enumerate group versions
ALL_FQ_APIS=(${ALL_FQ_APIS:-}) # e.g. k8s.io/kubernetes/pkg/apis/apps k8s.io/api/apps/v1
INT_FQ_APIS=(${INT_FQ_APIS:-}) # e.g. k8s.io/kubernetes/pkg/apis/apps
EXT_FQ_APIS=(${EXT_FQ_APIS:-}) # e.g. k8s.io/api/apps/v1
EXT_PB_APIS=(${EXT_PB_APIS:-}) # e.g. k8s.io/api/apps/v1

for GVs in ${GROUPS_WITH_VERSIONS}; do
  IFS=: read -r G Vs <<<"${GVs}"

  if [[ -n "${INT_APIS_PKG}" ]]; then
    ALL_FQ_APIS+=("${INT_APIS_PKG}/${G}")
    INT_FQ_APIS+=("${INT_APIS_PKG}/${G}")
  fi

  # enumerate versions
  for V in ${Vs//,/ }; do
    ALL_FQ_APIS+=("${EXT_APIS_PKG}/${G}/${V}")
    EXT_FQ_APIS+=("${EXT_APIS_PKG}/${G}/${V}")
  done
done

if [[ "${GENS}" = "all" ]] || [[ "${GENS}" = "all-external" ]] || grep -qw "deepcopy-external" <<<"${GENS}"; then
  echo "===========> Generating external deepcopy funcs"
  "${K8S_BIN}"/deepcopy-gen \
        --go-header-file /root/boilerplate.go.txt \
        --input-dirs "$(codegen_join , "${EXT_FQ_APIS[@]}")" \
        -O zz_generated.deepcopy \
        --bounding-dirs "${EXT_APIS_PKG}" \
        "$@"
fi

if [[ "${GENS}" = "all" ]] || [[ "${GENS}" = "all-external" ]] || grep -qw "client-external" <<<"${GENS}"; then
  echo "===========> Generating external clientset for ${GROUPS_WITH_VERSIONS} at ${OUTPUT_PKG}/clientset"
  "${K8S_BIN}"/client-gen \
        --go-header-file /root/boilerplate.go.txt \
        --clientset-name versioned \
        --input-base "" \
        --input "$(codegen_join , "${EXT_FQ_APIS[@]}")" \
        --output-package "${OUTPUT_PKG}"/clientset \
        "$@"
fi

if [[ "${GENS}" = "all" ]] || [[ "${GENS}" = "all-external" ]] || grep -qw "lister-external" <<<"${GENS}"; then
  echo "===========> Generating external listers for ${GROUPS_WITH_VERSIONS} at ${OUTPUT_PKG}/listers"
  "${K8S_BIN}"/lister-gen \
        --go-header-file /root/boilerplate.go.txt \
        --input-dirs "$(codegen_join , "${EXT_FQ_APIS[@]}")" \
        --output-package "${OUTPUT_PKG}"/listers \
        "$@"
fi

if [[ "${GENS}" = "all" ]] || [[ "${GENS}" = "all-external" ]] || grep -qw "informer-external" <<<"${GENS}"; then
  echo "===========> Generating external informers for ${GROUPS_WITH_VERSIONS} at ${OUTPUT_PKG}/informers"
  "${K8S_BIN}"/informer-gen \
        --go-header-file /root/boilerplate.go.txt \
        --input-dirs "$(codegen_join , "${EXT_FQ_APIS[@]}")" \
        --versioned-clientset-package "${OUTPUT_PKG}"/clientset/versioned \
        --listers-package "${OUTPUT_PKG}"/listers \
        --output-package "${OUTPUT_PKG}"/informers \
        "$@"
fi

if [[ "${GENS}" = "all" ]] || [[ "${GENS}" = "all-external" ]] || grep -qw "defaulter-external" <<<"${GENS}"; then
  echo "===========> Generating external defaulters"
  "${K8S_BIN}"/defaulter-gen \
        --go-header-file /root/boilerplate.go.txt \
        --input-dirs "$(codegen_join , "${EXT_FQ_APIS[@]}")" \
        -O zz_generated.defaults \
        "$@"
fi

if [[ "${GENS}" = "all" ]] || [[ "${GENS}" = "all-external" ]] || grep -qw "conversion-external" <<<"${GENS}"; then
  echo "===========> Generating external conversions"
  "${K8S_BIN}"/conversion-gen \
        --go-header-file /root/boilerplate.go.txt \
        --input-dirs "$(codegen_join , "${ALL_FQ_APIS[@]}")" \
        -O zz_generated.conversion \
        "$@"
fi