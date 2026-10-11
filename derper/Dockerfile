FROM --platform=$BUILDPLATFORM golang:alpine AS build
RUN apk add --no-cache ca-certificates
ARG DERPER_VERSION
RUN test -n "$DERPER_VERSION" || { echo "DERPER_VERSION build arg is required" >&2; exit 1; }
ARG TARGETOS TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go install -trimpath -ldflags="-s -w" tailscale.com/cmd/derper@${DERPER_VERSION} \
    && mkdir /out && find /go/bin -name derper -type f -exec mv {} /out/derper \;

FROM alpine:3
RUN apk add --no-cache ca-certificates
COPY --from=build /out/derper /usr/local/bin/derper
VOLUME /var/lib/derper
EXPOSE 80 443 3478/udp
ENTRYPOINT ["/usr/local/bin/derper"]
