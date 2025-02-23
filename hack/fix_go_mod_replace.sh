#!/bin/bash
set -eux -o pipefail

argocd_version="$(grep github.com/argoproj/argo-cd/v2 go.mod | awk '{print $2}')"
go get "github.com/argoproj/argo-cd/v2@${argocd_version}"

k8s_version="$(grep k8s.io/api go.mod | awk '{print $2}')"
perl -i -pne "s/(k8s.io\/\S+ => k8s.io\/\S+) .+$/\1 ${k8s_version}/g" go.mod
