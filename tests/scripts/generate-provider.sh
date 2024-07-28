#!/bin/sh

d=$(dirname "$0")

cd ${d}/../terraform-provider-crd

go mod init github.com/vvbogdanov87/terraform-provider-crd
tfpgen
go mod tidy
go install
cp .terraformrc ~/.terraformrc