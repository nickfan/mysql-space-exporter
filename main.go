package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	dbRows = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mysql_table_rows",
			Help: "Number of rows in table",
		},
		[]string{"database", "table"},
	)
	dbDataSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mysql_table_data_size_bytes",
			Help: "Data size of table in bytes",
		},
		[]string{"database", "table"},
	)
	dbIndexSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mysql_table_index_size_bytes",
			Help: "Index size of table in bytes",
		},
		[]string{"database", "table"},
	)
	dbDataFree = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mysql_table_data_free_bytes",
			Help: "Free space in table in bytes",
		},
		[]string{"database", "table"},
	)
	dbTotalSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mysql_table_total_size_bytes",
			Help: "Total size (data + index) of table in bytes",
		},
		[]string{"database", "table"},
	)
)

func init() {
	prometheus.MustRegister(dbRows)
	prometheus.MustRegister(dbDataSize)
	prometheus.MustRegister(dbIndexSize)
	prometheus.MustRegister(dbDataFree)
	prometheus.MustRegister(dbTotalSize)
}

func main() {
	dsn := os.Getenv("MYSQL_USER") + ":" + os.Getenv("MYSQL_PASSWORD") + "@tcp(" + os.Getenv("MYSQL_HOST") + ")/"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	go func() {
		for {
			collectMetrics(db)
			time.Sleep(60 * time.Second)
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":"+os.Getenv("EXPORTER_PORT"), nil)
}

func collectMetrics(db *sql.DB) {
	rows, err := db.Query(`
		SELECT 
			TABLE_SCHEMA,
			TABLE_NAME,
			TABLE_ROWS,
			DATA_LENGTH,
			INDEX_LENGTH,
			DATA_FREE,
			(DATA_LENGTH + INDEX_LENGTH) as TOTAL_SIZE
		FROM information_schema.tables
		WHERE TABLE_SCHEMA NOT IN ('mysql', 'information_schema', 'performance_schema')
	`)
	if err != nil {
		log.Printf("Error collecting metrics: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var schema, table string
		var tableRows, dataLength, indexLength, dataFree, totalSize float64
		
		if err := rows.Scan(&schema, &table, &tableRows, &dataLength, &indexLength, &dataFree, &totalSize); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		dbRows.WithLabelValues(schema, table).Set(tableRows)
		dbDataSize.WithLabelValues(schema, table).Set(dataLength)
		dbIndexSize.WithLabelValues(schema, table).Set(indexLength)
		dbDataFree.WithLabelValues(schema, table).Set(dataFree)
		dbTotalSize.WithLabelValues(schema, table).Set(totalSize)
	}
}