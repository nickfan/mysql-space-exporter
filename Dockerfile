FROM golang:1.20-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod init github.com/nickfan/mysql-space-exporter && \
    go get github.com/go-sql-driver/mysql && \
    go get github.com/prometheus/client_golang/prometheus && \
    go get github.com/prometheus/client_golang/prometheus/promhttp && \
    go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o mysql-space-exporter

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/mysql-space-exporter . 
EXPOSE 9104
ENTRYPOINT ["./mysql-space-exporter"]