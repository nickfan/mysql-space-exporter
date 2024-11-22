package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"
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
	enableLogging bool
	dbFilter     string
	sortField    string
	sortOrder    string
)

func init() {
	prometheus.MustRegister(dbRows)
	prometheus.MustRegister(dbDataSize)
	prometheus.MustRegister(dbIndexSize)
	prometheus.MustRegister(dbDataFree)
	prometheus.MustRegister(dbTotalSize)

	if val := os.Getenv("ENABLE_LOGGING"); val == "true" {
		enableLogging = true
	}

	dbFilter = os.Getenv("DB_FILTER")
	sortField = os.Getenv("SORT_FIELD")
	if sortField == "" {
		sortField = "TOTAL_SIZE"
	}

	sortOrder = os.Getenv("SORT_ORDER")
	if sortOrder == "" {
		sortOrder = "DESC"
	}
}

func main() {
	dsn := os.Getenv("MYSQL_USER") + ":" + os.Getenv("MYSQL_PASSWORD") + "@tcp(" + os.Getenv("MYSQL_HOST") + ")/"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	limit := 200
	if val, ok := os.LookupEnv("EXPORTER_LIMIT"); ok {
		if parsedVal, err := strconv.Atoi(val); err == nil {
			limit = parsedVal
		}
	}

	go func() {
		for {
			collectMetrics(db, limit)
			time.Sleep(60 * time.Second)
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":"+os.Getenv("EXPORTER_PORT"), nil)
}

func logError(format string, v ...interface{}) {
	if enableLogging {
		log.Printf(format, v...)
	}
}

const queryTemplate = `
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
	{{if .DBFilter}}AND TABLE_SCHEMA IN ({{.DBFilter}}){{end}}
	ORDER BY {{.SortField}} {{.SortOrder}}
	LIMIT ?
`

type queryParams struct {
	DBFilter  string
	SortField string
	SortOrder string
}

func buildQuery(params queryParams) (string, error) {
	tmpl, err := template.New("query").Parse(queryTemplate)
	if err != nil {
		return "", err
	}

	var query bytes.Buffer
	err = tmpl.Execute(&query, params)
	if err != nil {
		return "", err
	}

	return query.String(), nil
}

func collectMetrics(db *sql.DB, limit int) {
	var filterStr string
	if dbFilter != "" {
		quoted := make([]string, 0)
		for _, db := range strings.Split(dbFilter, ",") {
			quoted = append(quoted, fmt.Sprintf("'%s'", strings.TrimSpace(db)))
		}
		filterStr = strings.Join(quoted, ",")
	}

	params := queryParams{
		DBFilter:  filterStr,
		SortField: sortField,
		SortOrder: sortOrder,
	}

	query, err := buildQuery(params)
	if err != nil {
		logError("Error building query: %v", err)
		return
	}

	rows, err := db.Query(query, limit)
	if err != nil {
		logError("Error collecting metrics: %v", err)
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