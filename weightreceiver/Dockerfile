FROM golang:alpine AS builder
WORKDIR /build
COPY ./ ./
RUN go build

FROM scratch
COPY --from=0 ./build/weightreceiver ./build/weightreceiver
CMD ["/build/weightreceiver"]