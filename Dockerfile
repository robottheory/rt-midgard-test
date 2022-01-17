# Build Image
FROM golang:1.16-alpine AS build

# ca-certificates pull in default CA-s, without tihs https fetch from blockstore will fail
RUN apk add --no-cache make musl-dev gcc ca-certificates && update-ca-certificates

WORKDIR /tmp/midgard

# Cache Go dependencies like this:
COPY go.mod go.sum ./
RUN go mod download

COPY  . .

# Compile.
RUN CC=/usr/bin/gcc CGO_ENABLED=1 go build -v -a --ldflags '-linkmode external -extldflags=-static' -installsuffix cgo ./cmd/midgard
RUN CC=/usr/bin/gcc CGO_ENABLED=1 go build -v -a --ldflags '-linkmode external -extldflags=-static' -installsuffix cgo ./cmd/trimdb

# Main Image
FROM busybox

RUN mkdir -p openapi/generated
COPY --from=build /etc/ssl/certs /etc/ssl/certs
COPY --from=build /tmp/midgard/openapi/generated/doc.html ./openapi/generated/doc.html
COPY --from=build /tmp/midgard/midgard .
COPY --from=build /tmp/midgard/trimdb .
COPY config/config.json .
COPY resources /resources

CMD [ "./midgard", "config.json" ]