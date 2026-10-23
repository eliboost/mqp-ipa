FROM sdss-docker-local.repo.pnet.ch/pf/sdss/gitlab/dev/java21:5.9.0 AS builder

ARG MAVEN_SETTINGS_XML_PATH=./settings.xml

# check=skip=UndefinedVar
COPY --chown=$USER:$GROUP . /builds

WORKDIR /builds

RUN mvn -s ${MAVEN_SETTINGS_XML_PATH} verify

FROM middleware-docker-local.repo.pnet.ch/pf/mw/openjdk:1.0.202610220920-21-ubi9-minimal

RUN http_proxy=http://outapplpf.gslb.pnet.ch:3128 \
    https_proxy=http://outapplpf.gslb.pnet.ch:3128 \
    no_proxy=localhost,127.0.0.1,.pnet.ch,.pfin.ch

# hadolint ignore=DL3041
RUN microdnf update -y && microdnf install -y sudo iputils hostname findutils less bash curl-minimal glibc-langpack-en && microdnf clean all && rm -rf /var/cache/dnf

RUN useradd -ms /bin/bash artemis-monitor && usermod -aG wheel artemis-monitor && printf "artemis-monitor\tALL=(ALL)\tNOPASSWD: ALL" > /etc/sudoers.d/artemis-monitor

COPY --chown=artemis-monitor:artemis-monitor --from=builder "/builds/target/*.jar" /app.jar

# hadolint ignore=DL3066
USER artemis-monitor

EXPOSE 8080

# clear any shell overrides from the upstream image
ENTRYPOINT []

CMD ["java", "-server", "-jar", "/app.jar"]