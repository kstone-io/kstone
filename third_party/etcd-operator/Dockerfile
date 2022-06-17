FROM golang:1.12 as builder
MAINTAINER simotang "simotang@tencent.com"

ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

Add . /go/src/github.com/coreos/etcd-operator
WORKDIR /go/src/github.com/coreos/etcd-operator

RUN make


FROM alpine:3.6

COPY --from=builder /usr/share/zoneinfo/ /usr/share/zoneinfo/

ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

RUN apk add --no-cache ca-certificates

COPY --from=builder /go/src/github.com/coreos/etcd-operator/_output/bin/ /usr/local/bin/

RUN adduser -D etcd-operator
USER etcd-operator