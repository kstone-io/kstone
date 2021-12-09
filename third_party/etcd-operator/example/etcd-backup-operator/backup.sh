#!/bin/bash

timestamp=`date +%s`
echo $timestamp
cat ./backup_cr_qcloud.yaml | sed "s/TIMESTAMP/$timestamp/" | kubectl create -f -
