FROM linux-docker-local.repo.pnet.ch/pf/distroless/static:latest

ARG http_proxy=http://outapplpf.pnet.ch:3128
ARG https_proxy=http://outapplpf.pnet.ch:3128
ARG no_proxy=localhost,127.0.0.1,.pnet.ch
ARG TZ=Europe/Zurich

ENV TZ=${TZ}

ARG TARGETPLATFORM
COPY $TARGETPLATFORM/pf-activemq-operator /usr/bin/pf-activemq-operator

USER 65534

ENTRYPOINT ["/usr/bin/pf-activemq-operator"]