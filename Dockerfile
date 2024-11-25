FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o mysql-space-exporter

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/mysql-space-exporter .
EXPOSE 9104
ENTRYPOINT ["./mysql-space-exporter"]