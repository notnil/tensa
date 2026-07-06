FROM ubuntu:22.04

# Set the target architecture
ARG TARGETOS=linux
ARG TARGETARCH=arm64
ARG GO_VERSION=1.25.3

ENV DEBIAN_FRONTEND=noninteractive

# Base packages, toolchain, and audio development headers for CGO (oto/beep)
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates curl git build-essential pkg-config \
    binutils binutils-gold \
    iproute2 can-utils bluez dbus \
    libasound2 libasound2-dev \
    && rm -rf /var/lib/apt/lists/*

# Install Go (arm64) matching our desired version
RUN curl -fsSL https://go.dev/dl/go${GO_VERSION}.linux-arm64.tar.gz -o /tmp/go.tar.gz \
    && tar -C /usr/local -xzf /tmp/go.tar.gz \
    && rm /tmp/go.tar.gz

ENV PATH=/usr/local/go/bin:${PATH}
ENV CGO_ENABLED=1

WORKDIR /app

# Copy go.mod and go.sum first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the application for the target platform (single-stage image)
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-s -w" -o /app/tensactl ./cmd/tensactl

# Create logs directory
RUN mkdir -p /app/logs

# Default entrypoint and args
ENTRYPOINT ["/app/tensactl"]
CMD ["-c", "/app/cmd/tensactl/config.example.json"]