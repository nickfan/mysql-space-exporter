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

- `MYSQL_HOST`: MySQL主机地址 (默认: localhost:3306)
- `MYSQL_USER`: MySQL用户名
- `MYSQL_PASSWORD`: MySQL密码
- `EXPORTER_PORT`: 导出器端口 (默认: 9104)
- `EXPORTER_LIMIT`: 监控表数量限制 (默认: 200)
- `ENABLE_LOGGING`: 是否启用日志 (默认: false)
- `DB_FILTER`: 数据库过滤列表，逗号分隔
- `SORT_FIELD`: 排序字段 (默认: TOTAL_SIZE)
- `SORT_ORDER`: 排序方向 (默认: DESC)

## 快速开始

### 本地开发环境配置

### 安装
