ARG ARTEMIS_IMAGE_VERSION

FROM middleware-docker.repo.pnet.ch/arkmq-org/arkmq-org-broker-init:artemis.${ARTEMIS_IMAGE_VERSION}

    # renovate: datasource=artifactory depName=middleware-maven/ch/postfinance/activemq/pf-artemis-logging-plugin
ENV PFLOGGINGPLUGIN_VERSION=0.1.37  \
    # renovate: datasource=github-releases depName=prometheus/jmx_exporter
    JMX_PROMETHEUS_JAVAAGENT=1.5.0

USER 0

COPY prometheus /amq/prometheus

RUN mkdir -p /amq/lib/ \
    && curl -kfL https://repo.pnet.ch/artifactory/middleware-maven-releases-local/ch/postfinance/activemq/pf-artemis-logging-plugin/${PFLOGGINGPLUGIN_VERSION}/pf-artemis-logging-plugin-${PFLOGGINGPLUGIN_VERSION}.jar -o /amq/lib/pf-artemis-logging-plugin.jar \
    && curl -kfL https://repo.pnet.ch/artifactory/github-releases-remote/prometheus/jmx_exporter/releases/download/${JMX_PROMETHEUS_JAVAAGENT}/jmx_prometheus_javaagent-${JMX_PROMETHEUS_JAVAAGENT}.jar -o /amq/prometheus/jmx_prometheus_javaagent.jar

COPY scripts /amq/scripts

# hadolint ignore=DL3066
USER jboss