package main

import (
	"bufio"
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
	DotEnv        string
	Help          bool
	ServerPort    int // 新增 ServerPort 字段
}

func init() {
	prometheus.MustRegister(dbRows)
	prometheus.MustRegister(dbDataSize)
	prometheus.MustRegister(dbIndexSize)
	prometheus.MustRegister(dbDataFree)
	prometheus.MustRegister(dbTotalSize)
}

// 新增函数：显示帮助信息
func showHelp() {
	fmt.Println("MySQL Space Exporter")
	fmt.Println("\nUsage:")
	fmt.Println("  mysql-space-exporter [flags]")
	fmt.Println("\nFlags:")
	flag.PrintDefaults()
	os.Exit(0)
}

// 新增函数：读取环境变量文件
func loadEnvFile(envFile string) error {
	file, err := os.Open(envFile)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// 分割键值对
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		// 移除可能的引号
		value = strings.Trim(value, `"'`)
		os.Setenv(key, value)
	}
	return scanner.Err()
}

func parseConfig() *Config {
	config := &Config{}
	
	// 添加新的命令行参数
	flag.BoolVar(&config.Help, "help", false, "Show help information")
	flag.BoolVar(&config.Help, "h", false, "Show help information")
	flag.StringVar(&config.DotEnv, "dotenv", "", "Load environment variables from .env file")
	flag.StringVar(&config.DotEnv, "E", "", "Load environment variables from .env file")
	
	// 添加 server-port 参数
	flag.IntVar(&config.ServerPort, "server-port", getEnvAsIntDefault("SERVER_PORT", 9107), "Server port for metrics endpoint")
	
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

	// 处理帮助信息
	if config.Help {
		showHelp()
	}

	// 检查是否有任何参数
	if flag.NFlag() == 0 {
		// 检查 DB_HOST 环境变量
		if _, exists := os.LookupEnv("DB_HOST"); !exists {
			// 尝试加载 .env 文件
			if err := loadEnvFile(".env"); err != nil {
				showHelp()
			}
		}
	}

	// 如果指定了 dotenv 文件，则加载它
	if config.DotEnv != "" {
		envFile := config.DotEnv
		if envFile == "true" {
			envFile = ".env"
		}
		if err := loadEnvFile(envFile); err != nil {
			log.Printf("Warning: Could not load environment file: %v", err)
		}
	}
	
	return config
}

// 新增：根路径处理函数
func handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := `<html>
<head><title>MySQL Space Exporter</title></head>
<body>
<h1>MySQL Space Exporter</h1>
<p><a href="/metrics">Metrics</a></p>
</body>
</html>`
	w.Write([]byte(html))
}

func main() {
    config := parseConfig()

    // 构建DSN
    dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/", config.User, config.Password, config.Host, config.Port)
    db, err := sql.Open("mysql", dsn)
    if (err != nil) {
        log.Fatalf("Failed to open database connection: %v", err)
    }
    defer db.Close()

    // 测试数据库连接
    if err := db.Ping(); err != nil {
        log.Fatalf("Failed to ping database: %v", err)
    }
    log.Printf("Successfully connected to MySQL at %s:%d", config.Host, config.Port)

    // 立即执行一次metrics收集
    if err := collectMetrics(db, config); err != nil {
        log.Printf("Initial metrics collection failed: %v", err)
    }

    // 启动定期收集
    go func() {
        for {
            if err := collectMetrics(db, config); err != nil {
                log.Printf("Metrics collection failed: %v", err)
            }
            time.Sleep(60 * time.Second)
        }
    }()

    // 设置路由
    http.HandleFunc("/", handleRoot)
    http.Handle("/metrics", promhttp.Handler())

    // 启动服务器
    serverAddr := fmt.Sprintf(":%d", config.ServerPort)
    log.Printf("Starting server on %s", serverAddr)
    if err := http.ListenAndServe(serverAddr, nil); err != nil {
        log.Fatal(err)
    }
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

func logError(config *Config, format string, v ...interface{}) {
	if config.EnableLogging {
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

func collectMetrics(db *sql.DB, config *Config) error {
    // 测试连接是否有效
    if err := db.Ping(); err != nil {
        return fmt.Errorf("database ping failed: %v", err)
    }

    var filterStr string
    if config.DBFilter != "" {
        quoted := make([]string, 0)
        for _, db := range strings.Split(config.DBFilter, ",") {
            quoted = append(quoted, fmt.Sprintf("'%s'", strings.TrimSpace(db)))
        }
        filterStr = strings.Join(quoted, ",")
    }

    var tableFilterStr string
    if config.TableFilter != "" {
        quoted := make([]string, 0)
        for _, table := range strings.Split(config.TableFilter, ",") {
            quoted = append(quoted, fmt.Sprintf("'%s'", strings.TrimSpace(table)))
        }
        tableFilterStr = strings.Join(quoted, ",")
    }

    params := queryParams{
        DBFilter:    filterStr,
        TableFilter: tableFilterStr,
        SortField:   config.SortField,
        SortOrder:   config.SortOrder,
    }

    query, err := buildQuery(params)
    if err != nil {
        return fmt.Errorf("error building query: %v", err)
    }

    rows, err := db.Query(query, config.OutLimit)
    if err != nil {
        return fmt.Errorf("error executing query: %v", err)
    }
    defer rows.Close()

    // 记录处理的行数
    rowCount := 0

    for rows.Next() {
        var schema, table string
        var tableRows, dataLength, indexLength, dataFree, totalSize float64
        
        if err := rows.Scan(&schema, &table, &tableRows, &dataLength, &indexLength, &dataFree, &totalSize); err != nil {
            return fmt.Errorf("error scanning row: %v", err)
        }

        dbRows.WithLabelValues(schema, table).Set(tableRows)
        dbDataSize.WithLabelValues(schema, table).Set(dataLength)
        dbIndexSize.WithLabelValues(schema, table).Set(indexLength)
        dbDataFree.WithLabelValues(schema, table).Set(dataFree)
        dbTotalSize.WithLabelValues(schema, table).Set(totalSize)
        rowCount++
    }

    if config.EnableLogging {
        log.Printf("Successfully collected metrics for %d tables", rowCount)
    }

    return nil
}