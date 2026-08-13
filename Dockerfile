# syntax=docker/dockerfile:1.26@sha256:ecfaec9ed6d810b56388c508f4121597bfbba70d41a6dfeee4d8cad5f295fc32

# Build stage
FROM --platform=$BUILDPLATFORM golang:1.26 AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

# Cache dependencies.
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build.
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
  go build -trimpath -ldflags="-s -w" -o /claudelint ./cmd/claudelint

# Runtime stage
FROM gcr.io/distroless/static-debian12:nonroot

ARG TARGETARCH=amd64

COPY --from=builder /claudelint /claudelint

USER nonroot:nonroot

ENTRYPOINT ["/claudelint"]

