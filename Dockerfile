ARG DEBIAN_IMAGE=debian:stable-slim
ARG BASE=gcr.io/distroless/static-debian12:nonroot
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
FROM --platform=$BUILDPLATFORM golang:1.23 AS coredns-builder
COPY . /go/src/github.com/coredns/coredns
WORKDIR /go/src/github.com/coredns/coredns

RUN go env -w GO111MODULE=on && \
    go env -w GOPROXY=https://goproxy.cn,direct
#RUN  GOOS=${TARGETOS} GOARCH=${TARGETARCH}  go build github.com/coredns/coredns

RUN  #GOOS=${TARGETOS} GOARCH=${TARGETARCH} GOARM=${TARGETVARIANT#v}  go build -o coredns  .
RUN  GOOS=${TARGETOS} GOARCH=${TARGETARCH}   go build -o coredns  .

FROM  ${DEBIAN_IMAGE} AS build
SHELL [ "/bin/sh", "-ec" ]

RUN export DEBCONF_NONINTERACTIVE_SEEN=true \
           DEBIAN_FRONTEND=noninteractive \
           DEBIAN_PRIORITY=critical \
           TERM=linux ; \
    apt-get -qq update ; \
    apt-get -yyqq upgrade ; \
    apt-get -yyqq install ca-certificates libcap2-bin; \
    apt-get clean
COPY --from=coredns-builder /go/src/github.com/coredns/coredns/coredns /coredns
RUN setcap cap_net_bind_service=+ep /coredns

FROM  ${DEBIAN_IMAGE}
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /coredns /coredns
RUN export DEBCONF_NONINTERACTIVE_SEEN=true \
           DEBIAN_FRONTEND=noninteractive \
           DEBIAN_PRIORITY=critical \
           TERM=linux ; \
    apt-get -qq update ; \
    apt-get -yyqq upgrade ; \
    apt-get -yyqq install dnsutils; \
    apt-get clean
WORKDIR /
EXPOSE 53 53/udp
ENTRYPOINT ["/coredns"]
