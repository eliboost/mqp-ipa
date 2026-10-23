# extract layers into separate directories
FROM middleware-docker-local.repo.pnet.ch/pf/mw/openjdk:1.0.202610161422-21-rhel9 as builder
WORKDIR layers
COPY target/mqp-ops-ui-backend-exec.jar mqp-ops-ui-backend-exec.jar
RUN java -Djarmode=tools -jar mqp-ops-ui-backend-exec.jar extract --layers --launcher --destination extracted

# build actual image
FROM middleware-docker-local.repo.pnet.ch/pf/mw/openjdk:1.0.202610161422-21-rhel9

ENV PF_TRUSTSTORE="/var/opt/neviskeybox/default/pki-all/truststore.jks"

ARG APP_VERSION
LABEL ch.post.pf.mqpopsui.version=${APP_VERSION}

COPY --from=builder layers/extracted/dependencies/ /usr/share/mqp-ops-ui
COPY --from=builder layers/extracted/spring-boot-loader/ /usr/share/mqp-ops-ui
COPY --from=builder layers/extracted/snapshot-dependencies/ /usr/share/mqp-ops-ui
COPY --from=builder layers/extracted/project-dependencies/ /usr/share/mqp-ops-ui
COPY --from=builder layers/extracted/application/ /usr/share/mqp-ops-ui

WORKDIR /usr/share/mqp-ops-ui

ENTRYPOINT ["java", "org.springframework.boot.loader.launch.PropertiesLauncher"]