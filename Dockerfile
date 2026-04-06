FROM golang:1.21 AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -buildmode=c-shared -o /out_zerobus.so .

FROM fluent/fluent-bit:latest
COPY --from=builder /out_zerobus.so /fluent-bit/plugins/
COPY fluent-bit/plugins.conf /fluent-bit/etc/plugins.conf
COPY fluent-bit/fluent-bit.conf /fluent-bit/etc/fluent-bit.conf
