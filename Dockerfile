# gosx-site: a complete GoSX Studio website in one container.
#
#   docker build -t gosx-site .
#   docker run -p 8080:8080 -v gosx-site-data:/data \
#     -e GOSX_SITE_ADMIN_PASSWORD=change-me gosx-site
#
# Then open http://localhost:8080/admin and run the setup wizard. Everything
# the site owns — pages, pictures, messages — lives in the /data volume.
#
# The build drops the module's local `replace` directives, which point at
# sibling checkouts used for day-to-day development, and builds against the
# published m31labs.dev/gosx and gosx-admin versions go.mod already requires.

FROM golang:1.26-alpine AS build
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod edit -dropreplace m31labs.dev/gosx -dropreplace m31labs.dev/gosx-admin \
 && go mod download
COPY . .
RUN go mod edit -dropreplace m31labs.dev/gosx -dropreplace m31labs.dev/gosx-admin \
 && CGO_ENABLED=0 GOFLAGS=-mod=mod go build -trimpath -ldflags="-s -w" -o /out/gosx-site ./cmd/gosx-site

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata \
 && addgroup -S site && adduser -S -G site site \
 && mkdir -p /data && chown site:site /data
COPY --from=build /out/gosx-site /usr/local/bin/gosx-site
USER site
VOLUME ["/data"]
EXPOSE 8080 80 443
ENV GOSX_SITE_ADDR=0.0.0.0:8080 \
    GOSX_SITE_DATA=/data/site.db
# GOSX_SITE_ADMIN_PASSWORD is required: the binary refuses to listen on a
# non-loopback address without one, so an unprotected admin area can never
# be published by accident.
HEALTHCHECK --interval=30s --timeout=3s CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["gosx-site"]
