FROM golang:1.15-alpine AS dev

LABEL org.label-schema.vcs-url="https://github.com/MrSaints/forward-ext-authz-service" \
      maintainer="Ian L. <os@fyianlai.com>"

WORKDIR /forward-ext-authz-service/

RUN apk add --no-cache build-base curl

ENV GO111MODULE on
ENV GOPROXY https://proxy.golang.org

COPY go.mod go.sum /forward-ext-authz-service/

RUN go mod download


FROM dev as build

COPY ./ /forward-ext-authz-service/

RUN mkdir /build/

RUN CGO_ENABLED=0 \
    go build -v \
    -ldflags "-s" -a -installsuffix cgo \
    -o /build/forward-ext-authz-service \
    /forward-ext-authz-service/ \
    && chmod +x /build/forward-ext-authz-service


FROM alpine:3.12 AS prod

LABEL org.label-schema.vcs-url="https://github.com/MrSaints/forward-ext-authz-service" \
      maintainer="Ian L. <os@fyianlai.com>"

RUN apk add --no-cache bash ca-certificates curl jq wget nano

COPY --from=build /build/forward-ext-authz-service /forward-ext-authz-service/run

ARG BUILD_VERSION
ENV FORWARDEAZ_SERVICE_VERSION $BUILD_VERSION

ENTRYPOINT ["/forward-ext-authz-service/run"]
