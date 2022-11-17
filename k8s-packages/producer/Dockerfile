############################
# STEP 1 build executable binary
############################
#FROM golang:1.19-buster as builder
FROM golang:alpine3.16 as builder

# Install git + SSL ca certificates.
# Git is required for fetching the dependencies.  Make and protoc are needed to build server.
# Ca-certificates is required to call HTTPS endpoints.
RUN apk update && apk add git musl-dev gcc make ca-certificates && update-ca-certificates

#RUN dpkg --add-architecture amd64 \
#    && apt-get update \
#    && apt-get install -y --no-install-recommends gcc-x86-64-linux-gnu libc6-dev-amd64-cross

#RUN wget https://www.musl-libc.org/releases/musl-1.2.0.tar.gz && \
#   tar -xzf musl-1.2.0.tar.gz && \
#   cd musl-1.2.0 && \
#   ./configure --enable-static --disable-shared && \
#   make && make install

#RUN musl-gcc

# Create appuser
ENV USER=appuser
ENV UID=10001

# See https://stackoverflow.com/a/55757473/12429735
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/home/appuser" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"

WORKDIR /src/mypackage/myapp/

COPY config config
COPY internal internal
COPY openapi openapi
COPY go.mod .
COPY go.sum .

RUN go mod tidy

# We do this so docker will cache everything above this line, including module downloads
COPY cmd/producer cmd/producer

RUN go mod tidy

# Build the binary
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags='-w -s -extldflags "-static"' -a \
          -o /go/bin/producer /src/mypackage/myapp/cmd/producer

#RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 CC=x86_64-linux-gnu-gcc go build -ldflags='-w -s -extldflags "-static"' -a \
#          -o /go/bin/producer /src/mypackage/myapp/cmd/producer


############################
# STEP 2 build a small image
############################
FROM scratch

# Import from builder.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Copy our static executable
COPY --from=builder /go/bin/producer /go/bin/producer

# Use an unprivileged user.
# RUN mkdir /home/appuser && chown -R appuser:appuser /home/appuser
USER appuser:appuser

ENV MIDGARD_TENDERMINTURL="bla"
# Run the hello binary.
ENTRYPOINT ["/go/bin/producer"]
