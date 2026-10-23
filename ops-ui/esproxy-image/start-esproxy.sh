#!/bin/bash
#
#
# Copyright 2009 - 2023 by PostFinance AG - all rights reserved
#
#

### Warning: when editing this file, be sure to set Unix line endings!

sed -e "s/ESPROXY_TARGET_SERVER_FRONTEND/${ESPROXY_TARGET_SERVER_FRONTEND}/g" -e "s/ESPROXY_TARGET_SERVER_BACKEND/${ESPROXY_TARGET_SERVER_BACKEND}/g" /appl/config/esproxy-config.xml.template > /appl/config/esproxy-config.xml

export MIDW_JVM_OPTIONS_APPLICATION="-Djdk.tls.acknowledgeCloseNotify=true -Daplat.asesp.bootstrap.configfile=${ESP_CONFIGFILE} -Dorg.apache.xml.security.ignoreLineBreaks=true -Djavax.net.ssl.trustStore=${APLAT_ESPROXY_SSL_TRUSTSTORE}"

echo "Starting ES Proxy (userId=[$(id)], PID=$$) with following args: PF_TC_DIR=${PF_TC_DIR} ESPROXY_TARGET_SERVER_FRONTEND=${ESPROXY_TARGET_SERVER_FRONTEND} ESPROXY_TARGET_SERVER_BACKEND=${ESPROXY_TARGET_SERVER_BACKEND}"
exec "${PF_TC_DIR}/bin/pf-StartTomcat.sh"
