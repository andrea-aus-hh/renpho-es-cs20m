FROM golang:alpine AS builder
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o weightreceiver .

FROM alpine:latest
WORKDIR /
COPY --from=builder /build/weightreceiver .
RUN chmod +x ./weightreceiver
CMD ["./weightreceiver"]
