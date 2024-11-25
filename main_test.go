package main

import (
	"database/sql"
	"flag"
	"os"
	"strings"
	"testing"
)

func TestGetEnvDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		want         string
		shouldSetEnv bool
	}{
		{
			name:         "使用默认值",
			key:         "TEST_KEY",
			defaultValue: "default",
			want:        "default",
			shouldSetEnv: false,
		},
		{
			name:         "使用环境变量值",
			key:         "TEST_KEY",
			defaultValue: "default",
			envValue:    "fromenv",
			want:        "fromenv",
			shouldSetEnv: true,
		},
		{
			name:         "存在的环境变量",
			key:          "TEST_VAR",
			defaultValue: "default",
			envValue:     "test",
			want:         "test",
			shouldSetEnv: true,
		},
		{
			name:         "不存在的环境变量",
			key:          "NON_EXISTING_VAR",
			defaultValue: "default",
			want:         "default",
			shouldSetEnv: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldSetEnv && tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			if got := getEnvDefault(tt.key, tt.defaultValue); got != tt.want {
				t.Errorf("getEnvDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildQuery(t *testing.T) {
	tests := []struct {
		name        string
		params      queryParams
		wantErr     bool
		checkResult func(string) bool
	}{
		{
			name: "无过滤条件",
			params: queryParams{
				DBFilter:    "",
				TableFilter: "",
				SortField:   "TOTAL_SIZE",
				SortOrder:   "DESC",
			},
			wantErr: false,
			checkResult: func(query string) bool {
				return len(query) > 0 && 
					!contains(query, "TABLE_SCHEMA IN") &&
					!contains(query, "TABLE_NAME IN")
			},
		},
		{
			name: "包含数据库过滤",
			params: queryParams{
				DBFilter:    "'test_db'",
				TableFilter: "",
				SortField:   "TOTAL_SIZE",
				SortOrder:   "DESC",
			},
			wantErr: false,
			checkResult: func(query string) bool {
				return contains(query, "TABLE_SCHEMA IN ('test_db')")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildQuery(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.checkResult(got) {
				t.Errorf("buildQuery() 生成的查询不符合预期: %v", got)
			}
		})
	}
}

func TestCollectMetrics(t *testing.T) {
	db, err := sql.Open("mysql", "root:@tcp(localhost:3306)/")
	if (err != nil) {
		t.Fatal(err)
	}
	defer db.Close()

	// 创建测试配置
	testConfig := &Config{
		OutLimit:  10,
		SortField: "TOTAL_SIZE",
		SortOrder: "DESC",
	}

	// 使用正确的参数类型调用
	collectMetrics(db, testConfig)
	// 由于是无效连接，不会panic即为通过
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && s != substr && strings.Contains(s, substr)
}

func TestConfigParsing(t *testing.T) {
	// 测试命令行参数
	tests := []struct {
		name     string
		args     []string
		envVars  map[string]string
		expected Config
	}{
		{
			name: "Default values",
			args: []string{},
			expected: Config{
				Host:          "localhost",
				Port:          3306,
				User:          "root",
				Password:      "",
				OutLimit:      200,
				SortField:     "TOTAL_SIZE",
				SortOrder:     "DESC",
				EnableLogging: false,
			},
		},
		{
			name: "Long format arguments",
			args: []string{
				"--host=testhost",
				"--port=3307",
				"--user=testuser",
				"--password=testpass",
				"--limit=100",
			},
			expected: Config{
				Host:          "testhost",
				Port:          3307,
				User:          "testuser",
				Password:      "testpass",
				OutLimit:      100,
				SortField:     "TOTAL_SIZE",
				SortOrder:     "DESC",
				EnableLogging: false,
			},
		},
		{
			name: "Short format arguments",
			args: []string{
				"-H", "testhost",
				"-P", "3307",
				"-u", "testuser",
				"-p", "testpass",
			},
			expected: Config{
				Host:          "testhost",
				Port:          3307,
				User:          "testuser",
				Password:      "testpass",
				OutLimit:      200,
				SortField:     "TOTAL_SIZE",
				SortOrder:     "DESC",
				EnableLogging: false,
			},
		},
		{
			name: "Environment variables",
			envVars: map[string]string{
				"DB_HOST":        "envhost",
				"DB_PORT":        "3308",
				"DB_USER":        "envuser",
				"DB_PASSWD":      "envpass",
				"OUT_LIMIT":      "150",
				"SORT_FIELD":     "DATA_SIZE",
				"SORT_ORDER":     "ASC",
				"ENABLE_LOGGING": "true",
			},
			expected: Config{
				Host:          "envhost",
				Port:          3308,
				User:          "envuser",
				Password:      "envpass",
				OutLimit:      150,
				SortField:     "DATA_SIZE",
				SortOrder:     "ASC",
				EnableLogging: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 重置标志
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
			
			// 设置环境变量
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			// 设置命令行参数
			oldArgs := os.Args
			os.Args = append([]string{"cmd"}, tt.args...)
			defer func() { os.Args = oldArgs }()

			// 创建新的配置实例
			config := parseConfig()

			// 验证结果
			if config.Host != tt.expected.Host {
				t.Errorf("Host = %v, want %v", config.Host, tt.expected.Host)
			}
			if config.Port != tt.expected.Port {
				t.Errorf("Port = %v, want %v", config.Port, tt.expected.Port)
			}
			if config.User != tt.expected.User {
				t.Errorf("User = %v, want %v", config.User, tt.expected.User)
			}
			if config.Password != tt.expected.Password {
				t.Errorf("Password = %v, want %v", config.Password, tt.expected.Password)
			}
			if config.OutLimit != tt.expected.OutLimit {
				t.Errorf("OutLimit = %v, want %v", config.OutLimit, tt.expected.OutLimit)
			}
			if config.SortField != tt.expected.SortField {
				t.Errorf("SortField = %v, want %v", config.SortField, tt.expected.SortField)
			}
			if config.SortOrder != tt.expected.SortOrder {
				t.Errorf("SortOrder = %v, want %v", config.SortOrder, tt.expected.SortOrder)
			}
			if config.EnableLogging != tt.expected.EnableLogging {
				t.Errorf("EnableLogging = %v, want %v", config.EnableLogging, tt.expected.EnableLogging)
			}
		})
	}
}
