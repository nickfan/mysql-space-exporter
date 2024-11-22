# MySQL Space Exporter

MySQL Space Exporter 是一个用于监控 MySQL 数据库表空间使用情况的 Prometheus exporter。它能够帮助您实时监控数据库中各个表的空间占用、行数等关键指标。

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

### 使用 Docker
