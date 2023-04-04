FROM golang:latest AS builder

# Install dependencies
RUN apt-get update \
    && apt-get install -y \
    libheif-dev \
    libwebp-dev

# Install go dependencies
WORKDIR /go/src/towebp
COPY go.mod .
COPY go.sum .
RUN go mod download

# Build
COPY main.go .
RUN go build -ldflags="-extldflags=-static" -o /go/bin/towebp

# Run
FROM gcr.io/distroless/static-debian11
COPY --from=builder /go/bin/towebp /
ENTRYPOINT ["/towebp"]
