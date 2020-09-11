# Build Image
FROM golang:1.15 AS build

ARG pg_host
ARG rpc_host
ARG thornode_host

ENV PG_HOST=$pg_host
ENV RPC_HOST=$rpc_host
ENV THORNODE_HOST=$thornode_host

RUN env

# Install jq to update the chain service config.
RUN curl -sS https://dl.yarnpkg.com/debian/pubkey.gpg | apt-key add -
RUN echo "deb https://dl.yarnpkg.com/debian/ stable main" | tee /etc/apt/sources.list.d/yarn.list
RUN apt-get update
RUN apt-get install -y jq apt-utils make yarn


WORKDIR /tmp/midgard

# Cache Go dependencies like this:
COPY go.mod go.sum ./
RUN go mod download

COPY  . .

# Compile.
RUN CGO_ENABLED=0 GOOS=linux go build -v -a -installsuffix cgo ./cmd/midgard

# Generate config.
RUN mkdir -p /etc/midgard
RUN cat ./cmd/midgard/config.json | jq \
  --arg RPC_HOST "$RPC_HOST" \
  --arg THORNODE_HOST "$THORNODE_HOST" \
  --arg PG_HOST "$PG_HOST" \
  '.timescale["host"] = $PG_HOST | \
  .thorchain["rpc_host"] = $RPC_HOST | \
  .thorchain["host"] = $THORNODE_HOST' > /etc/midgard/config.json

# Prints password too 🚨
RUN cat /etc/midgard/config.json


# Main Image
FROM scratch

COPY --from=build /etc/midgard/config.json .
COPY --from=build /tmp/midgard/midgard .

CMD [ "./midgard", "config.json" ]
