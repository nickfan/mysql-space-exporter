FROM golang:1.19-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o mysql-space-exporter

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/mysql-space-exporter .
EXPOSE 9104
ENTRYPOINT ["./mysql-space-exporter"]