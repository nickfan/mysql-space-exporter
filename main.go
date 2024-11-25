package main

import (
	"bytes"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com.prometheus/client_golang/prometheus/promhttp"
	_ "github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus"
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

type Config struct {
	Host          string
	Port          int
	User          string
	Password      string
	DBFilter      string
	TableFilter   string
	OutLimit      int
	SortField     string
	SortOrder     string
	EnableLogging bool
}

func init() {
	prometheus.MustRegister(dbRows)
	prometheus.MustRegister(dbDataSize)
	prometheus.MustRegister(dbIndexSize)
	prometheus.MustRegister(dbDataFree)
	prometheus.MustRegister(dbTotalSize)
}

func main() {
	config := &Config{}
	
	// 长参数
	flag.StringVar(&config.Host, "host", getEnvDefault("DB_HOST", "localhost"), "Database host")
	flag.IntVar(&config.Port, "port", getEnvAsIntDefault("DB_PORT", 3306), "Database port")
	flag.StringVar(&config.User, "user", getEnvDefault("DB_USER", "root"), "Database user")
	flag.StringVar(&config.Password, "password", getEnvDefault("DB_PASSWD", ""), "Database password")
	flag.StringVar(&config.DBFilter, "db-filter", getEnvDefault("DB_FILTER", ""), "Database filter")
	flag.StringVar(&config.TableFilter, "table-filter", getEnvDefault("TABLE_FILTER", ""), "Table filter")
	flag.IntVar(&config.OutLimit, "limit", getEnvAsIntDefault("OUT_LIMIT", 200), "Output limit")
	flag.StringVar(&config.SortField, "sort-field", getEnvDefault("SORT_FIELD", "TOTAL_SIZE"), "Sort field")
	flag.StringVar(&config.SortOrder, "sort-order", getEnvDefault("SORT_ORDER", "DESC"), "Sort order")
	flag.BoolVar(&config.EnableLogging, "enable-logging", getEnvAsBoolDefault("ENABLE_LOGGING", false), "Enable logging")

	// 短参数
	flag.StringVar(&config.Host, "H", getEnvDefault("DB_HOST", "localhost"), "Database host")
	flag.StringVar(&config.User, "u", getEnvDefault("DB_USER", "root"), "Database user")
	flag.StringVar(&config.Password, "p", getEnvDefault("DB_PASSWD", ""), "Database password")
	flag.IntVar(&config.Port, "P", getEnvAsIntDefault("DB_PORT", 3306), "Database port")
	
	flag.Parse()

	dsn := config.User + ":" + config.Password + "@tcp(" + config.Host + ")/"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	go func() {
		for {
			collectMetrics(db, config.OutLimit)
			time.Sleep(60 * time.Second)
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":"+strconv.Itoa(config.Port), nil)
}

func getEnvDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsIntDefault(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvAsBoolDefault(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
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
	{{if .TableFilter}}AND TABLE_NAME IN ({{.TableFilter}}){{end}}
	ORDER BY {{.SortField}} {{.SortOrder}}
	LIMIT ?
`

type queryParams struct {
	DBFilter    string
	TableFilter string
	SortField   string
	SortOrder   string
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

	var tableFilterStr string
	if tableFilter != "" {
		quoted := make([]string, 0)
		for _, table := range strings.Split(tableFilter, ",") {
			quoted = append(quoted, fmt.Sprintf("'%s'", strings.TrimSpace(table)))
		}
		tableFilterStr = strings.Join(quoted, ",")
	}

	params := queryParams{
		DBFilter:    filterStr,
		TableFilter: tableFilterStr,
		SortField:   sortField,
		SortOrder:   sortOrder,
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