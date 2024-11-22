
#!/bin/bash
set -e

echo "Initializing Go module..."
rm -f go.mod go.sum
go mod init github.com/nickfan/mysql-space-exporter
go get github.com/go-sql-driver/mysql
go get github.com/prometheus/client_golang/prometheus
go get github.com/prometheus/client_golang/prometheus/promhttp
go mod tidy

echo "Go module setup complete!"