FROM golang:alpine AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /build
COPY ./ ./
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o weightreceiver .

FROM scratch
COPY --from=builder /build/weightreceiver /build/weightreceiver
CMD ["/build/weightreceiver"]