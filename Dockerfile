FROM bash:latest

ARG TARGETARCH="amd64"
ARG TARGETOS="linux"
ARG TARGETPLATFORM="linux/amd64"

ARG USER_UID=1001

ADD https://github.com/juev/starred/releases/latest/download/starred-linux-amd64 /usr/local/bin/starred

RUN set -eux; \
    adduser -D runner -u $USER_UID; \
    chmod +rx /usr/local/bin/starred;

USER runner