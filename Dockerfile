FROM golang:alpine AS builder
WORKDIR /app
ENV CGO_ENABLED 0
COPY . /app
RUN go build -o htm ./cmd/http/multi/main.go

FROM scratch

COPY --from=builder /app/htm /

EXPOSE 8000
expose 8829/udp

CMD ["/htm"]
