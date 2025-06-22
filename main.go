package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

type Config struct {
	ESURL    string
	Index    string
	Username string
	Password string
	Output   string
	Input    string
	Mode     string
}

func main() {
	// 定义命令行参数
	var config Config
	flag.StringVar(&config.ESURL, "url", "http://localhost:9200", "Elasticsearch URL")
	flag.StringVar(&config.Index, "index", "", "Elasticsearch index name")
	flag.StringVar(&config.Username, "username", "", "Elasticsearch username")
	flag.StringVar(&config.Password, "password", "", "Elasticsearch password")
	flag.StringVar(&config.Output, "output", "output.jsonl", "Output JSONL file path (for export mode)")
	flag.StringVar(&config.Input, "input", "", "Input JSONL file path (for import mode)")
	flag.StringVar(&config.Mode, "mode", "export", "Operation mode: export or import")
	flag.Parse()

	if config.Index == "" {
		log.Fatal("Index name is required")
	}

	// 验证模式参数
	if config.Mode != "export" && config.Mode != "import" {
		log.Fatal("Mode must be either 'export' or 'import'")
	}

	// 验证模式相关的必需参数
	if config.Mode == "import" && config.Input == "" {
		log.Fatal("Input file path is required for import mode")
	}

	// 初始化 Elasticsearch 客户端
	es, err := initESClient(config)
	if err != nil {
		log.Fatalf("Failed to initialize Elasticsearch client: %v", err)
	}

	// 根据模式执行相应操作
	switch config.Mode {
	case "export":
		if err := exportIndexData(context.Background(), es, config); err != nil {
			log.Fatalf("Failed to export data: %v", err)
		}
		fmt.Printf("Data exported successfully to %s\n", config.Output)
	case "import":
		if err := importIndexData(context.Background(), es, config); err != nil {
			log.Fatalf("Failed to import data: %v", err)
		}
		fmt.Printf("Data imported successfully from %s\n", config.Input)
	}
}

func initESClient(config Config) (*elasticsearch.Client, error) {
	cfg := elasticsearch.Config{
		Addresses: []string{config.ESURL},
		Username:  config.Username,
		Password:  config.Password,
	}

	return elasticsearch.NewClient(cfg)
}

func exportIndexData(ctx context.Context, es *elasticsearch.Client, config Config) error {
	// 打开输出文件
	file, err := os.Create(config.Output)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer file.Close()

	// 创建带缓冲的写入器
	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// 初始化 Scroll 查询
	var scrollID string
	batchSize := 1000
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"match_all": map[string]interface{}{},
		},
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return fmt.Errorf("failed to encode query: %v", err)
	}

	totalDocs := 0

	for {
		var res *esapi.Response
		if scrollID == "" {
			// 第一次查询
			res, err = es.Search(
				es.Search.WithContext(ctx),
				es.Search.WithIndex(config.Index),
				es.Search.WithBody(&buf),
				es.Search.WithScroll(2*time.Minute),
				es.Search.WithSize(batchSize),
			)
		} else {
			// 后续 Scroll 查询
			res, err = es.Scroll(
				es.Scroll.WithContext(ctx),
				es.Scroll.WithScrollID(scrollID),
				es.Scroll.WithScroll(2*time.Minute),
			)
		}

		if err != nil {
			return fmt.Errorf("scroll request failed: %v", err)
		}
		defer res.Body.Close()

		if res.IsError() {
			return fmt.Errorf("scroll response error: %s", res.String())
		}

		// 解析响应
		var response struct {
			ScrollID string `json:"_scroll_id"`
			Hits     struct {
				Hits []struct {
					Source json.RawMessage `json:"_source"`
				} `json:"hits"`
			} `json:"hits"`
		}

		if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
			return fmt.Errorf("failed to decode response: %v", err)
		}

		// 更新 Scroll ID
		scrollID = response.ScrollID

		// 处理当前批次数据
		if len(response.Hits.Hits) == 0 {
			break // 没有更多数据
		}

		for _, hit := range response.Hits.Hits {
			// 写入一行 JSON 数据
			if _, err := writer.Write(hit.Source); err != nil {
				return fmt.Errorf("failed to write to file: %v", err)
			}
			// 添加换行符
			if _, err := writer.Write([]byte("\n")); err != nil {
				return fmt.Errorf("failed to write newline: %v", err)
			}

			totalDocs++
		}

		fmt.Printf("Processed %d documents...\n", totalDocs)
	}

	// 清理 Scroll
	if scrollID != "" {
		_, err := es.ClearScroll(
			es.ClearScroll.WithContext(ctx),
			es.ClearScroll.WithScrollID(scrollID),
		)
		if err != nil {
			log.Printf("Warning: failed to clear scroll: %v", err)
		}
	}

	return nil
}

func importIndexData(ctx context.Context, es *elasticsearch.Client, config Config) error {
	// 打开输入文件
	file, err := os.Open(config.Input)
	if err != nil {
		return fmt.Errorf("failed to open input file: %v", err)
	}
	defer file.Close()

	// 创建带缓冲的读取器
	scanner := bufio.NewScanner(file)

	// 读取所有文档
	var documents []json.RawMessage
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		// 跳过空行
		if line == "" {
			continue
		}

		// 解析每行的 JSON
		var doc json.RawMessage
		if err := json.Unmarshal([]byte(line), &doc); err != nil {
			return fmt.Errorf("failed to parse JSON at line %d: %v", err)
		}
		documents = append(documents, doc)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	fmt.Printf("Found %d documents to import\n", len(documents))

	// 批量导入文档
	batchSize := 1000
	totalImported := 0

	for i := 0; i < len(documents); i += batchSize {
		end := i + batchSize
		if end > len(documents) {
			end = len(documents)
		}

		batch := documents[i:end]
		if err := importBatch(ctx, es, config.Index, batch); err != nil {
			return fmt.Errorf("failed to import batch %d-%d: %v", i, end-1, err)
		}

		totalImported += len(batch)
		fmt.Printf("Imported %d/%d documents...\n", totalImported, len(documents))
	}

	return nil
}

func importBatch(ctx context.Context, es *elasticsearch.Client, index string, documents []json.RawMessage) error {
	// 构建批量请求
	var buf bytes.Buffer

	for _, doc := range documents {
		// 添加索引操作
		indexAction := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": index,
			},
		}

		if err := json.NewEncoder(&buf).Encode(indexAction); err != nil {
			return fmt.Errorf("failed to encode index action: %v", err)
		}

		// 添加文档内容
		if err := json.NewEncoder(&buf).Encode(json.RawMessage(doc)); err != nil {
			return fmt.Errorf("failed to encode document: %v", err)
		}
	}

	// 执行批量请求
	res, err := es.Bulk(
		bytes.NewReader(buf.Bytes()),
		es.Bulk.WithContext(ctx),
		es.Bulk.WithIndex(index),
	)
	if err != nil {
		return fmt.Errorf("bulk request failed: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("bulk response error: %s, body: %s", res.String(), string(body))
	}

	// 检查批量操作结果
	var result struct {
		Errors bool `json:"errors"`
		Items  []struct {
			Index struct {
				Status int    `json:"status"`
				Error  string `json:"error,omitempty"`
			} `json:"index"`
		} `json:"items"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode bulk response: %v", err)
	}

	if result.Errors {
		var errors []string
		for i, item := range result.Items {
			if item.Index.Status >= 400 {
				errors = append(errors, fmt.Sprintf("document %d: %s", i, item.Index.Error))
			}
		}
		if len(errors) > 0 {
			return fmt.Errorf("bulk import errors: %v", errors)
		}
	}

	return nil
}
