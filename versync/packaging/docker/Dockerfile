FROM linux-docker-local.repo.pnet.ch/pf/alpine:3.24.2202610140623


ARG TARGETARCH
ARG TARGETOS
ARG http_proxy=http://outapplpf.pnet.ch:3128
ARG https_proxy=http://outapplpf.pnet.ch:3128
ARG no_proxy=localhost,127.0.0.1,.pnet.ch
ARG TZ=Europe/Zurich

ENV TZ=${TZ}

ARG TARGETPLATFORM
COPY $TARGETPLATFORM/versync /usr/bin/versync
COPY $TARGETPLATFORM/versync /usr/bin/versync

USER 65534

ENTRYPOINT ["/usr/bin/versync"]