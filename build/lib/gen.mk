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

# set the kubernetes apimachinery package dir
K8S_APIMACHINERY_DIR = $(shell go list -f '{{ .Dir }}' -m k8s.io/apimachinery)
# set the kubernetes api package dir
K8S_API_DIR = $(shell go list -f '{{ .Dir }}' -m k8s.io/api)
# set the gogo protobuf package dir
GOGO_PROTOBUF_DIR = $(shell go list -f '{{ .Dir }}' -m github.com/gogo/protobuf)
EXT_PB_APIS = "k8s.io/api/core/v1 k8s.io/api/apps/v1"
# set the code generator image version
CODE_GENERATOR_VERSION := v1.21.3

.PHONY: gen.run
gen.run: gen.api

# ==============================================================================
# Generator

.PHONY: gen.api
gen.api:
	@$(DOCKER) run -it --rm \
		-v $(ROOT_DIR):/go/src/$(ROOT_PACKAGE) \
		-e EXT_PB_APIS=$(EXT_PB_APIS)\
	 	$(REGISTRY_PREFIX)/code-generator:$(CODE_GENERATOR_VERSION) \
	 	/root/code.sh \
	 	all \
	 	$(ROOT_PACKAGE)/pkg/generated \
	 	$(ROOT_PACKAGE)/pkg/apis \
	 	$(ROOT_PACKAGE)/pkg/apis \
		"kstone:v1alpha1"