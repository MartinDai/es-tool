# ES Tool - Elasticsearch 数据导入导出工具

这是一个用于 Elasticsearch 数据导入和导出的命令行工具，支持两种运行模式：导出模式和导入模式。

## 功能特性

- **导出模式**: 从 Elasticsearch 索引导出数据到 JSONL 文件
- **导入模式**: 从 JSONL 文件导入数据到 Elasticsearch 索引
- 支持批量处理，提高性能
- 支持认证
- 支持自定义 Elasticsearch URL

## 安装

### 方法一：使用 Makefile（推荐）

确保已安装 Go 1.16 或更高版本，然后运行：

```bash
# 安装依赖
make deps

# 构建当前平台的可执行文件
make build

# 或者构建所有平台的可执行文件
make build-all
```

### 方法二：直接使用 Go 命令

```bash
go mod tidy
go build -o es-tool main.go
```

## 构建选项

### 使用 Makefile

```bash
# 查看所有可用命令
make help

# 构建当前平台
make build

# 构建所有支持的平台
make build-all

# 构建特定平台
make build-linux-amd64    # Linux AMD64
make build-linux-arm64    # Linux ARM64
make build-darwin-amd64   # macOS AMD64
make build-darwin-arm64   # macOS ARM64
make build-windows-amd64  # Windows AMD64

# 创建发布包（包含所有平台）
make release

# 清理构建文件
make clean
```

### 支持的平台

- Linux: AMD64, ARM64, 386, ARM
- macOS: AMD64, ARM64
- Windows: AMD64, 386

## 使用方法

### 导出模式

从 Elasticsearch 索引导出数据到 JSONL 文件：

```bash
./es-tool -mode=export -index=your_index_name -output=exported_data.jsonl
```

### 导入模式

从 JSONL 文件导入数据到 Elasticsearch 索引：

```bash
./es-tool -mode=import -index=your_index_name -input=import_data.jsonl
```

## 命令行参数

| 参数 | 类型 | 必需 | 默认值 | 描述 |
|------|------|------|--------|------|
| `-url` | string | 否 | `http://localhost:9200` | Elasticsearch 服务器 URL |
| `-index` | string | 是 | - | Elasticsearch 索引名称 |
| `-username` | string | 否 | - | Elasticsearch 用户名 |
| `-password` | string | 否 | - | Elasticsearch 密码 |
| `-mode` | string | 否 | `export` | 运行模式：`export` 或 `import` |
| `-output` | string | 否 | `output.jsonl` | 导出模式下的输出文件路径 |
| `-input` | string | 否 | - | 导入模式下的输入文件路径 |

## 使用示例

### 基本导出

```bash
./es-tool -index=products -output=products_export.jsonl
```

### 带认证的导出

```bash
./es-tool -url=https://your-es-cluster:9200 -index=products -username=elastic -password=your_password -output=products_export.jsonl
```

### 基本导入

```bash
./es-tool -mode=import -index=products -input=products_import.jsonl
```

### 带认证的导入

```bash
./es-tool -mode=import -url=https://your-es-cluster:9200 -index=products -username=elastic -password=your_password -input=products_import.jsonl
```

## 数据格式

### 导出格式（JSONL）

导出的文件格式为 JSONL（JSON Lines），每行一个 JSON 对象：

```jsonl
{"id": 1, "name": "Product 1", "price": 100}
{"id": 2, "name": "Product 2", "price": 200}
{"id": 3, "name": "Product 3", "price": 300}
```

### 导入格式（JSONL）

导入的文件必须使用 JSONL 格式，每行一个 JSON 对象。空行会被自动跳过。

### JSONL 格式的优势

1. **流式处理**: 可以逐行处理，不需要将整个文件加载到内存
2. **易于追加**: 可以轻松向文件末尾添加新数据
3. **兼容性好**: 大多数数据处理工具都支持 JSONL 格式
4. **错误定位**: 出错时可以精确定位到具体的行号

## 注意事项

1. **索引存在性**: 导入模式下，目标索引必须已经存在
2. **批量处理**: 工具使用批量操作提高性能，默认批次大小为 1000 个文档
3. **内存使用**: 导入模式下会逐行读取 JSONL 文件，内存使用更高效
4. **错误处理**: 导入过程中如果出现错误，会显示详细的错误信息和行号
5. **认证**: 如果 Elasticsearch 启用了安全认证，请提供用户名和密码
6. **文件格式**: 确保导入文件使用 UTF-8 编码

## 错误排查

### 常见错误

1. **连接错误**: 检查 Elasticsearch URL 是否正确
2. **认证错误**: 检查用户名和密码是否正确
3. **索引不存在**: 确保目标索引已创建
4. **权限错误**: 确保用户有足够的权限访问索引
5. **JSON 格式错误**: 检查 JSONL 文件中的 JSON 格式是否正确

### 调试模式

可以通过查看详细的错误输出来诊断问题：

```bash
./es-tool -mode=import -index=test -input=data.jsonl 2>&1 | tee debug.log
```

### 验证 JSONL 文件格式

可以使用以下命令验证 JSONL 文件格式：

```bash
# 检查每行是否为有效的 JSON
while IFS= read -r line; do
    if [ -n "$line" ]; then
        echo "$line" | jq . > /dev/null || echo "Invalid JSON: $line"
    fi
done < your_file.jsonl
```
