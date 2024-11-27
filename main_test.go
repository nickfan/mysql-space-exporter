package main

import (
	"database/sql"
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
    // 保存原始的命令行参数和环境变量
    originalArgs := os.Args
    originalEnv := os.Environ()
    
    // 备份并移除现有的 .env 文件（如果存在）
    if _, err := os.Stat(".env"); err == nil {
        if err := os.Rename(".env", ".env.backup"); err != nil {
            t.Fatalf("Failed to backup .env file: %v", err)
        }
        defer func() {
            if err := os.Rename(".env.backup", ".env"); err != nil {
                t.Errorf("Failed to restore .env file: %v", err)
            }
        }()
    }
    
    // 清理函数
    cleanup := func() {
        os.Args = originalArgs
        os.Clearenv()
        for _, env := range originalEnv {
            parts := strings.SplitN(env, "=", 2)
            if len(parts) == 2 {
                os.Setenv(parts[0], parts[1])
            }
        }
    }
    
    tests := []struct {
        name     string
        args     []string
        envVars  map[string]string
        validate func(*testing.T, *Config)
    }{
        {
            name: "Default values",
            args: []string{"cmd"},
            envVars: map[string]string{}, // 确保没有环境变量
            validate: func(t *testing.T, c *Config) {
                if c.Host != "localhost" {
                    t.Errorf("Host = %v, want localhost", c.Host)
                }
                if c.Port != 3306 {
                    t.Errorf("Port = %v, want 3306", c.Port)
                }
                if c.User != "root" {
                    t.Errorf("User = %v, want root", c.User)
                }
                if c.OutLimit != 200 {
                    t.Errorf("OutLimit = %v, want 200", c.OutLimit)
                }
            },
        },
        {
            name: "Environment variables",
            args: []string{"cmd"},
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
            validate: func(t *testing.T, c *Config) {
                if c.Host != "envhost" {
                    t.Errorf("Host = %v, want envhost", c.Host)
                }
                if c.Port != 3308 {
                    t.Errorf("Port = %v, want 3308", c.Port)
                }
                if c.User != "envuser" {
                    t.Errorf("User = %v, want envuser", c.User)
                }
                if c.Password != "envpass" {
                    t.Errorf("Password = %v, want envpass", c.Password)
                }
            },
        },
        {
            name: "Command line arguments",
            args: []string{
                "cmd",
                "--host=testhost",
                "--port=3307",
                "--user=testuser",
                "--password=testpass",
            },
            validate: func(t *testing.T, c *Config) {
                if c.Host != "testhost" {
                    t.Errorf("Host = %v, want testhost", c.Host)
                }
                if c.Port != 3307 {
                    t.Errorf("Port = %v, want 3307", c.Port)
                }
                if c.User != "testuser" {
                    t.Errorf("User = %v, want testuser", c.User)
                }
                if c.Password != "testpass" {
                    t.Errorf("Password = %v, want testpass", c.Password)
                }
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // 每个测试前清理环境
            cleanup()
            
            // 完全清除环境变量
            os.Clearenv()
            
            // 设置环境变量
            for k, v := range tt.envVars {
                if err := os.Setenv(k, v); err != nil {
                    t.Fatalf("Failed to set environment variable %s: %v", k, err)
                }
            }
            
            // 设置命令行参数
            os.Args = tt.args
            
            // 执行测试
            config := parseConfig()
            
            // 验证结果
            tt.validate(t, config)
        })
    }

    // 测试完成后恢复环境
    cleanup()
}
