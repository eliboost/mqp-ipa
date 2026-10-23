#!/bin/bash

echo "#### The config dir locates at ${CONFIG_INSTANCE_DIR} ####"

echo "#### Copying pf-artemis-logging-plugin.jar to ${CONFIG_INSTANCE_DIR}/lib ####"
cp /amq/lib/pf-artemis-logging-plugin.jar "${CONFIG_INSTANCE_DIR}/lib/"

echo "#### Copying jmx_prometheus_javaagent.jar to ${CONFIG_INSTANCE_DIR}/lib/ ####"
cp /amq/prometheus/jmx_prometheus_javaagent.jar "${CONFIG_INSTANCE_DIR}/lib/"

echo "#### Copying jmx-config.yaml to ${CONFIG_INSTANCE_DIR}/etc/ ####"
cp /amq/prometheus/jmx-config.yaml "${CONFIG_INSTANCE_DIR}/etc/"

echo "#### Append JAVA_ARGS for JMX Exporter Java agent to ${CONFIG_INSTANCE_DIR}/etc/artemis.profile ####"
# We want this to output $ARTEMIS_INSTANCE without expansion
# shellcheck disable=SC2016
echo 'JAVA_ARGS=" $JAVA_ARGS -javaagent:$ARTEMIS_INSTANCE/lib/jmx_prometheus_javaagent.jar=8175:$ARTEMIS_INSTANCE/etc/jmx-config.yaml "' >> "${CONFIG_INSTANCE_DIR}/etc/artemis.profile"
echo "" >> "${CONFIG_INSTANCE_DIR}/etc/artemis.profile"

echo "#### Custom config done. ####"
