package timeseries

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/datacenter/internal/message"
)

type Config struct {
	RESTAddr       string `yaml:"rest_addr"`  // http://localhost:6041
	BatchSize      int    `yaml:"batch_size"`
	FlushInterval  int    `yaml:"flush_interval"`
	BufferCapacity int    `yaml:"buffer_capacity"`
}

// WriterStats 写入器统计信息
type WriterStats struct {
	MessagesWritten int64 `json:"messages_written"`
	BufferSize      int   `json:"buffer_size"`
	BufferCapacity  int   `json:"buffer_capacity"`
}

type Writer struct {
	config          Config
	httpClient      *http.Client
	buffer          chan *message.DeviceEnvelope
	quit            chan struct{}
	wg              sync.WaitGroup
	messagesWritten int64
}

func NewWriter(config Config) *Writer {
	return &Writer{
		config:     config,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		buffer:     make(chan *message.DeviceEnvelope, config.BufferCapacity),
		quit:       make(chan struct{}),
	}
}

func (w *Writer) Start() error {
	if err := w.initDB(); err != nil {
		return fmt.Errorf("failed to init DB: %w", err)
	}
	w.wg.Add(1)
	go w.flushLoop()
	fmt.Println("[TDengine] writer started (REST)")
	return nil
}

func (w *Writer) Stop() error {
	close(w.quit)
	w.wg.Wait()
	fmt.Println("[TDengine] writer stopped")
	return nil
}

func (w *Writer) Write(env *message.DeviceEnvelope) error {
	select {
	case w.buffer <- env:
		return nil
	default:
		return fmt.Errorf("buffer full")
	}
}

func (w *Writer) execSQL(sql string) error {
	req, _ := http.NewRequest("POST", w.config.RESTAddr+"/rest/sql", strings.NewReader(sql))
	req.Header.Set("Content-Type", "text/plain")
	req.SetBasicAuth("root", "taosdata")
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	body := string(respBody)
	// TDengine REST API 返回 JSON，code!=0 表示错误
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}
	if strings.Contains(body, `"code":`) && !strings.Contains(body, `"code":0,`) {
		return fmt.Errorf("TDengine: %s", body)
	}
	return nil
}

func (w *Writer) initDB() error {
	if err := w.execSQL("CREATE DATABASE IF NOT EXISTS iot_data KEEP 90"); err != nil {
		return fmt.Errorf("create database failed: %w", err)
	}
	fmt.Println("[TDengine] database iot_data created/verified")

	// 先删除旧表（列名可能不同），再重建
	w.execSQL("DROP STABLE IF EXISTS iot_data.device_data")
	if err := w.execSQL("CREATE STABLE iot_data.device_data (ts TIMESTAMP, val BINARY(512), quality INT) TAGS (device_id BINARY(64), domain_id BINARY(64), topic_name BINARY(128))"); err != nil {
		return fmt.Errorf("create stable failed: %w", err)
	}
	fmt.Println("[TDengine] stable device_data created/verified")
	return nil
}

func (w *Writer) flushLoop() {
	defer w.wg.Done()
	ticker := time.NewTicker(time.Duration(w.config.FlushInterval) * time.Second)
	defer ticker.Stop()
	batch := make([]*message.DeviceEnvelope, 0, w.config.BatchSize)

	for {
		select {
		case <-w.quit:
			if len(batch) > 0 {
				w.flush(batch)
			}
			return
		case env := <-w.buffer:
			batch = append(batch, env)
			if len(batch) >= w.config.BatchSize {
				w.flush(batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				w.flush(batch)
				batch = batch[:0]
			}
		}
	}
}

// GetStats 返回写入器统计信息
func (w *Writer) GetStats() WriterStats {
	return WriterStats{
		MessagesWritten: atomic.LoadInt64(&w.messagesWritten),
		BufferSize:      len(w.buffer),
		BufferCapacity:  w.config.BufferCapacity,
	}
}

func (w *Writer) flush(batch []*message.DeviceEnvelope) {
	for _, env := range batch {
		for _, unit := range env.Units {
			tableName := fmt.Sprintf("device_%s", env.DeviceID)
			ts := time.UnixMilli(unit.Timestamp).Format("2006-01-02 15:04:05.000")
			payload := string(unit.Payload)

			// 先确保子表存在
			createSQL := fmt.Sprintf(
				"CREATE TABLE IF NOT EXISTS iot_data.%s USING iot_data.device_data TAGS ('%s','%s','%s')",
				tableName, env.DeviceID, env.DomainID, unit.Topic,
			)
			if err := w.execSQL(createSQL); err != nil {
				fmt.Printf("[TDengine] create table error for %s: %v\n", tableName, err)
				continue
			}

			// 插入数据
			insertSQL := fmt.Sprintf(
				"INSERT INTO iot_data.%s VALUES ('%s','%s',0)",
				tableName, ts, payload,
			)
			if err := w.execSQL(insertSQL); err != nil {
				fmt.Printf("[TDengine] insert error for %s: %v\n", tableName, err)
			} else {
				atomic.AddInt64(&w.messagesWritten, 1)
			}
		}
	}
}
