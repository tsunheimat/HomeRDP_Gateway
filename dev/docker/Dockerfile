FROM golang:1.24 AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -trimpath -o /out/rdpgw ./cmd/rdpgw && \
    go build -trimpath -o /out/rdpgw-auth ./cmd/auth

FROM debian:bookworm-slim

# Install runtime dependencies and OpenSSL for local certificate generation.
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates openssl tzdata && \
    rm -rf /var/lib/apt/lists/* && \
    update-ca-certificates

# add user
RUN adduser --disabled-password --gecos "" --home /opt/rdpgw --uid 1001 rdpgw

# certificate generation
RUN mkdir -p /opt/rdpgw && cd /opt/rdpgw && \
    random=$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 32 | head -n 1) && \
    openssl genrsa -des3 -passout pass:$random -out server.pass.key 2048 && \
    openssl rsa -passin pass:$random -in server.pass.key -out key.pem && \
    rm server.pass.key && \
    openssl req -new -sha256 -key key.pem -out server.csr \
    -subj "/C=US/ST=VA/L=SomeCity/O=MyCompany/OU=MyDivision/CN=rdpgw" && \
    openssl x509 -req -days 365 -in server.csr -signkey key.pem -out server.pem

# make tempdir in case filestore is used
ADD dev/docker/tmp.tar /

COPY --chown=1001 dev/docker/run.sh /run.sh
COPY --chown=1001 dev/docker/run.lib.sh /run.lib.sh
COPY --from=builder --chown=1001 /out/rdpgw /opt/rdpgw/rdpgw
COPY --from=builder --chown=1001 /out/rdpgw-auth /opt/rdpgw/rdpgw-auth
COPY --chown=1001 cmd/rdpgw/templates /opt/rdpgw/templates
COPY --chown=1001 assets /opt/rdpgw/assets

RUN chmod +x /opt/rdpgw/rdpgw && \
    chmod +x /opt/rdpgw/rdpgw-auth && \
    chmod u+s /opt/rdpgw/rdpgw-auth

USER 0
WORKDIR /opt/rdpgw
ENTRYPOINT ["/bin/sh", "/run.sh"]
