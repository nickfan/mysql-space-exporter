# MySQL Space Exporter

MySQL表空间监控的Prometheus exporter。

## 参数说明

### 命令行参数

| 参数 | 短参数 | 环境变量 | 默认值 | 说明 |
|------|--------|----------|--------|------|
| --help | -h | - | false | 显示帮助信息 |
| --host | -H | DB_HOST | localhost | 数据库主机地址 |
| --port | -P | DB_PORT | 3306 | 数据库端口 |
| --user | -u | DB_USER | root | 数据库用户名 |
| --password | -p | DB_PASSWD | - | 数据库密码 |
| --server-port | - | SERVER_PORT | 9107 | Exporter服务端口 |
| --db-filter | - | DB_FILTER | - | 数据库过滤列表（逗号分隔） |
| --table-filter | - | TABLE_FILTER | - | 表名过滤列表（逗号分隔） |
| --limit | - | OUT_LIMIT | 200 | 输出记录数限制 |
| --sort-field | - | SORT_FIELD | TOTAL_SIZE | 排序字段 |
| --sort-order | - | SORT_ORDER | DESC | 排序方向(ASC/DESC) |
| --enable-logging | - | ENABLE_LOGGING | false | 启用日志记录 |
| --dotenv | -E | - | - | 加载指定的环境变量文件 |

## 功能特性

- 监控 MySQL 表空间使用情况
- 支持监控多个关键指标：
  - 表行数
  - 数据大小
  - 索引大小
  - 碎片空间大小
  - 总空间大小（数据+索引）
- 自动按表大小排序
- 可配置监控表的数量限制
- 支持 Docker 容器化部署
- 兼容 Prometheus 监控规范

## 环境变量配置

- `DB_HOST`: MySQL主机地址 (默认: localhost)
- `DB_PORT`: MySQL主机端口 (默认: 3306)
- `DB_USER`: MySQL用户名
- `DB_PASSWD`: MySQL密码
- `SERVER_PORT`: 导出器端口 (默认: 9107)
- `ENABLE_LOGGING`: 是否启用日志 (默认: false)
- `OUT_LIMIT`: 监控表数量限制 (默认: 200)
- `SORT_FIELD`: 排序字段 (默认: TOTAL_SIZE)
- `SORT_ORDER`: 排序方向 (默认: DESC)
- `DB_FILTER`: 数据库过滤列表，逗号分隔
- `TABLE_FILTER`: 数据库表过滤列表，逗号分隔

## 快速开始

### 安装

首次根据需求调整配置

```sh
cat >.env <<EOF
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWD=secret
SERVER_PORT=9107
ENABLE_LOGGING=false
OUT_LIMIT=200
SORT_FIELD=TOTAL_SIZE
SORT_ORDER=DESC
DB_FILTER=
TABLE_FILTER=

EOF

```

docker-compose.yml配置：

```sh
cat >docker-compose.yml <<EOF
version: '3'
services:
  mysql-space-exporter:
    env_file: .env
    image: nickfan/mysql-space-exporter:latest
    container_name: mysql-space-exporter
    ports:
      - "${SERVER_PORT}:9107"

EOF

```

启动服务：

```sh
docker compose up -d

```

验证效果：

```sh
curl http://localhost:9107/metrics

```

prometheus采集配置：

*简单配置模式示例*

```yml
scrape_configs:
  - job_name: 'mysql-space-exporter'
    static_configs:
      - targets: ['localhost:9107']
        labels:
          env: 'local'
          instance: 'localhost'
```


*多环境配置模式示例*

```yml
scrape_configs:
  # 统一使用相同的 job_name，通过 labels 区分环境
  - job_name: 'mysql-space-exporter'
    static_configs:
      - targets: ['dev-001.myhost.internal:9107']
        labels:
          env: 'dev'
          instance: 'dev-001'
      - targets: ['test-001.myhost.internal:9107']
        labels:
          env: 'test'
          instance: 'test-001'
      - targets: ['prod-001.myhost.internal:9107']
        labels:
          env: 'prod'
          instance: 'prod-001'

```

*服务发现模式示例*

采集配置：
```yml
  - job_name: 'mysql-space-exporter'
    honor_labels: true
    file_sd_configs:
      - files: [ '/etc/prometheus/sd_config/mysql-space-exporter/*.yml' ]
        refresh_interval: 3m

```

实例配置，比如：/etc/prometheus/sd_config/mysql-space-exporter/instance.yml

```yml
- labels:
    job_name: mysql-space-exporter
    idc: aliyun
    region: cn-hangzhou
    project: myproject
    env: prod
    instance: prod-001
  targets:
    - prod-001.myhost.internal:9107

```