FROM golang:1.19 as builder
ENV GOOS linux
ENV CGO_ENABLED 0
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY *.go *.go.yml ./
RUN go build -o labelflared

FROM alpine:3.17 as runner
RUN apk add --no-cache ca-certificates
COPY --from=builder /build/labelflared .
CMD ./labelflared
