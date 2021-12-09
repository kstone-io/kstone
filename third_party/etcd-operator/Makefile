DEFAULT_GOAL := build-all
build-all:
	curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
	./hack/update_vendor.sh
	./hack/build/operator/build
	./hack/build/backup-operator/build
	./hack/build/restore-operator/build 

dev:
	@docker build --network=host -t ccr.ccs.tencentyun.com/ccs-dev/etcd-operator:dev .
	@docker push ccr.ccs.tencentyun.com/ccs-dev/etcd-operator:dev