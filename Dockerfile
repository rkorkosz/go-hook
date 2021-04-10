FROM golang:alpine AS builder
WORKDIR /app
ENV CGO_ENABLED 0
COPY . /app
RUN go build .

FROM scratch

COPY --from=builder /app/go-hook /

EXPOSE 8000
expose 8829/udp

CMD ["/go-hook"]
