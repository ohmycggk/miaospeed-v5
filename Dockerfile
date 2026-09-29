# syntax=docker/dockerfile:1

FROM golang:1.26-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# miaospeed embeds a few private assets at compile time (see README).
# Generate functional equivalents in-tree:
#   - root CA bundle      -> reuse the builder image's system bundle
#   - miaoko self-signed  -> fresh throwaway self-signed cert (used by -mtls)
#   - BUILDTOKEN.key      -> from build secret `buildtoken`, else the koipy default
#   - default js scripts  -> committed in engine/embeded/
RUN set -e; \
    cp /etc/ssl/certs/ca-certificates.crt preconfigs/embeded/ca-certificates.crt; \
    mkdir -p preconfigs/embeded/miaokoCA; \
    openssl req -x509 -newkey rsa:2048 -nodes \
        -keyout preconfigs/embeded/miaokoCA/miaoko.key \
        -out preconfigs/embeded/miaokoCA/miaoko.crt \
        -days 3650 -subj "/CN=miaospeed"; \
    chmod 600 preconfigs/embeded/miaokoCA/miaoko.key

RUN --mount=type=secret,id=buildtoken,required=false \
    set -e; \
    if [ -s /run/secrets/buildtoken ]; then \
        cp /run/secrets/buildtoken utils/embeded/BUILDTOKEN.key; \
    else \
        printf '%s\n' 'MIAOKO4|580JxAo049R|GEnERAl|1X571R930|T0kEN' > utils/embeded/BUILDTOKEN.key; \
    fi

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG COMMIT=unknown
ARG BUILDCOUNT=0
ARG BRAND=miaospeed
RUN set -e; \
    COMPILATIONTIME=$(date -u +%Y-%m-%dT%H:%M:%SZ); \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath \
        -ldflags "-s -w -X main.COMMIT=${COMMIT} -X main.BUILDCOUNT=${BUILDCOUNT} -X main.BRAND=${BRAND} -X main.COMPILATIONTIME=${COMPILATIONTIME}" \
        -o /out/miaospeed .

FROM alpine:3

RUN apk add --no-cache ca-certificates tzdata

COPY --from=build /out/miaospeed /usr/local/bin/miaospeed
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

ENV TZ=Asia/Shanghai
EXPOSE 9966

ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["server"]
