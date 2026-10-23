#!/bin/bash

# renovate: datasource=github-tags depName=fybrik/crdoc
export CRDOC_VERSION=v0.6.4
# renovate: datasource=gitlab-tags depName=middleware/activemq/pf-activemq-operator
export PFACTIVEMQ_CRD_VERSION=v0.17.7

mkdir -p tmp && cd tmp || exit
curl --fail -L https://repo.pnet.ch/artifactory/github-releases-remote/fybrik/crdoc/releases/download/$CRDOC_VERSION/crdoc_Linux_x86_64.tar.gz -O
tar -xvf crdoc_Linux_x86_64.tar.gz && rm crdoc_Linux_x86_64.tar.gz && chmod +x crdoc
curl --fail https://gitlab.pnet.ch/middleware/activemq/pf-activemq-operator/-/raw/$PFACTIVEMQ_CRD_VERSION/config/crd/bases/pf-activemq-operator.pnet.ch_pfactivemqs.yaml -O

echo "generate documentation"
./crdoc --resources ./*.yaml --output ../docs/reference/APIDOC.md
cat ../docs/reference/APIDOC.md >> ../docs/reference/api_reference.md
rm ../docs/reference/APIDOC.md