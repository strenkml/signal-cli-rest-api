# x86_64: official GraalVM native binary from AsamK/signal-cli GitHub releases
ARG SIGNAL_CLI_VERSION=0.14.2
# aarch64: pre-built package from @morph027 — update independently when a new version is packaged
ARG SIGNAL_CLI_NATIVE_PACKAGE_VERSION=0.14.1+morph027+2

ARG SWAG_VERSION=1.16.4

ARG BUILD_VERSION_ARG=unset

FROM golang:1.26-trixie AS buildcontainer

ARG SIGNAL_CLI_VERSION
ARG SIGNAL_CLI_NATIVE_PACKAGE_VERSION
ARG SWAG_VERSION
ARG BUILD_VERSION_ARG

RUN dpkg-reconfigure debconf --frontend=noninteractive \
	&& apt-get update \
	&& apt-get -y install --no-install-recommends wget curl gnupg \
	&& rm -rf /var/lib/apt/lists/*

# Download the signal-cli native binary for the target architecture.
# x86_64: official release from AsamK/signal-cli GitHub releases.
# aarch64: pre-built package from @morph027 (https://packaging.gitlab.io/signal-cli/).
RUN arch="$(uname -m)"; \
    case "$arch" in \
        x86_64) \
            wget -nv https://github.com/AsamK/signal-cli/releases/download/v${SIGNAL_CLI_VERSION}/signal-cli-${SIGNAL_CLI_VERSION}-Linux-native.tar.gz -O /tmp/signal-cli-native.tar.gz \
            && mkdir -p /tmp/signal-cli-native \
            && tar xf /tmp/signal-cli-native.tar.gz -C /tmp/signal-cli-native \
            && mv /tmp/signal-cli-native/signal-cli /usr/bin/signal-cli-native ;; \
        aarch64) \
            curl -fsSL https://packaging.gitlab.io/signal-cli/gpg.key | gpg -o /usr/share/keyrings/signal-cli-native.pgp --dearmor \
            && echo "deb [signed-by=/usr/share/keyrings/signal-cli-native.pgp] https://packaging.gitlab.io/signal-cli signalcli main" > /etc/apt/sources.list.d/morph027-signal-cli.list \
            && mkdir -p /tmp/signal-cli-native \
            && cd /tmp/signal-cli-native \
            && apt-get update \
            && apt-get download signal-cli-native=${SIGNAL_CLI_NATIVE_PACKAGE_VERSION} \
            && ar x *.deb \
            && tar xf data.tar.gz \
            && mv /tmp/signal-cli-native/usr/bin/signal-cli-native /usr/bin/signal-cli-native ;; \
        *) echo "Unsupported architecture: $arch" && exit 1 ;; \
    esac;

RUN go install github.com/swaggo/swag/cmd/swag@v${SWAG_VERSION}

COPY src/api /tmp/signal-cli-rest-api-src/api
COPY src/client /tmp/signal-cli-rest-api-src/client
COPY src/datastructs /tmp/signal-cli-rest-api-src/datastructs
COPY src/storage /tmp/signal-cli-rest-api-src/storage
COPY src/utils /tmp/signal-cli-rest-api-src/utils
COPY src/scripts /tmp/signal-cli-rest-api-src/scripts
COPY src/main.go /tmp/signal-cli-rest-api-src/
COPY src/go.mod /tmp/signal-cli-rest-api-src/
COPY src/go.sum /tmp/signal-cli-rest-api-src/
COPY src/plugin_loader.go /tmp/signal-cli-rest-api-src/

RUN cd /tmp/signal-cli-rest-api-src && ${GOPATH}/bin/swag init
RUN cd /tmp/signal-cli-rest-api-src && go build -o signal-cli-rest-api main.go
RUN cd /tmp/signal-cli-rest-api-src && go test ./client -v && go test ./utils -v
RUN cd /tmp/signal-cli-rest-api-src/scripts && go build -o jsonrpc2-helper
RUN cd /tmp/signal-cli-rest-api-src && go build -buildmode=plugin -o signal-cli-rest-api_plugin_loader.so plugin_loader.go

# Runtime container

FROM ubuntu:noble

ENV GIN_MODE=release
ENV PORT=8080

ARG BUILD_VERSION_ARG
ENV BUILD_VERSION=$BUILD_VERSION_ARG
ENV SIGNAL_CLI_REST_API_PLUGIN_SHARED_OBJ_DIR=/usr/bin/

RUN dpkg-reconfigure debconf --frontend=noninteractive \
	&& apt-get update \
	&& apt-get install -y --no-install-recommends util-linux supervisor curl locales \
	&& rm -rf /var/lib/apt/lists/*

RUN sed -i -e 's/# en_US.UTF-8 UTF-8/en_US.UTF-8 UTF-8/' /etc/locale.gen && \
    dpkg-reconfigure --frontend=noninteractive locales && \
    update-locale LANG=en_US.UTF-8

ENV LANG en_US.UTF-8

COPY --from=buildcontainer /usr/bin/signal-cli-native /usr/bin/signal-cli-native
COPY --from=buildcontainer /tmp/signal-cli-rest-api-src/signal-cli-rest-api /usr/bin/signal-cli-rest-api
COPY --from=buildcontainer /tmp/signal-cli-rest-api-src/scripts/jsonrpc2-helper /usr/bin/jsonrpc2-helper
COPY --from=buildcontainer /tmp/signal-cli-rest-api-src/signal-cli-rest-api_plugin_loader.so /usr/bin/signal-cli-rest-api_plugin_loader.so
COPY entrypoint.sh /entrypoint.sh

RUN userdel ubuntu -r \
	&& groupadd -g 1000 signal-api \
	&& useradd --no-log-init -M -d /home -s /bin/bash -u 1000 -g 1000 signal-api \
	&& mkdir -p /signal-cli-config/ \
	&& mkdir -p /home/.local/share/signal-cli

EXPOSE ${PORT}

ENV SIGNAL_CLI_CONFIG_DIR=/home/.local/share/signal-cli
ENV SIGNAL_CLI_UID=1000
ENV SIGNAL_CLI_GID=1000
ENV SIGNAL_CLI_CHOWN_ON_STARTUP=true

ENTRYPOINT ["/entrypoint.sh"]

HEALTHCHECK --interval=20s --timeout=10s --retries=3 \
    CMD curl -f http://localhost:${PORT}/v1/health || exit 1
