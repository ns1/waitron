FROM golang:1.15-buster as builder

ENV GOPATH=/opt/go

RUN mkdir -p /opt/go/src/github.com/ns1/waitron
COPY . /opt/go/src/github.com/ns1/waitron/
RUN cd /opt/go/src/github.com/ns1/waitron \
    && go build -o bin/waitron . \
    && mv bin/waitron /usr/local/bin/waitron

FROM debian:buster-slim
# Install some basic tools for use in build commands.
RUN apt-get -y update && apt-get -y install wget curl ipmitool strace openssh-client iputils-ping dnsutils httpie iptables
COPY --from=builder /usr/local/bin/waitron /usr/local/bin/waitron

ENTRYPOINT [ "waitron", "--config", "/etc/waitron/config.yml"]

HEALTHCHECK --interval=10s --timeout=5s --start-period=30s CMD curl -X GET http://localhost/health
