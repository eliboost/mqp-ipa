ARG ARTEMIS_IMAGE_VERSION

FROM middleware-docker.repo.pnet.ch/arkmq-org/activemq-artemis-broker-kubernetes:artemis.${ARTEMIS_IMAGE_VERSION}

USER 0

RUN curl -kfL https://repo.pnet.ch:443/artifactory/middleware-generic-local/test/artemis/artemis-server-2.42.0.jar \
    -o artemis-server-2.42.0.jar && mv artemis-server-2.42.0.jar /opt/amq/artemis-server-2.42.0.jar

USER 185