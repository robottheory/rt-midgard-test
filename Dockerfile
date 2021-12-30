# Build Image
FROM golang:1.16-alpine AS build

RUN apk add --no-cache make musl-dev gcc

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
COPY --from=build /tmp/midgard/openapi/generated/doc.html ./openapi/generated/doc.html
COPY --from=build /tmp/midgard/midgard .
COPY --from=build /tmp/midgard/trimdb .
COPY config/config.json .

CMD [ "./midgard", "config.json" ]