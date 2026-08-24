package timeseries

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/datacenter/internal/message"
	_ "github.com/taosdata/driver-go/v3/taosRestful"
)

type Config struct {
	DSN            string `yaml:"dsn"`            // root:taosdata@tcp(localhost:6030)/
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
	db              *sql.DB
	buffer          chan *message.DeviceEnvelope
	quit            chan struct{}
	wg              sync.WaitGroup
	messagesWritten int64
}

func NewWriter(config Config) *Writer {
	return &Writer{
		config: config,
		buffer: make(chan *message.DeviceEnvelope, config.BufferCapacity),
		quit:   make(chan struct{}),
	}
}

func (w *Writer) Start() error {
	db, err := sql.Open("taosRestful", w.config.DSN)
	if err != nil {
		return fmt.Errorf("failed to open TDengine: %w", err)
	}
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping TDengine: %w", err)
	}
	w.db = db

	if err := w.initDB(); err != nil {
		return fmt.Errorf("failed to init DB: %w", err)
	}
	w.wg.Add(1)
	go w.flushLoop()
	fmt.Println("[TDengine] writer started (native)")
	return nil
}

func (w *Writer) Stop() error {
	close(w.quit)
	w.wg.Wait()
	if w.db != nil {
		w.db.Close()
	}
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

func (w *Writer) initDB() error {
	if _, err := w.db.Exec("CREATE DATABASE IF NOT EXISTS iot_data KEEP 90"); err != nil {
		return fmt.Errorf("create database failed: %w", err)
	}
	fmt.Println("[TDengine] database iot_data created/verified")

	if _, err := w.db.Exec("CREATE STABLE IF NOT EXISTS iot_data.device_data (ts TIMESTAMP, val BINARY(512), quality INT) TAGS (device_id BINARY(64), domain_id BINARY(64), topic_name BINARY(128))"); err != nil {
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
	fmt.Printf("[TDengine] flushing batch of %d messages\n", len(batch))
	for _, env := range batch {
		for _, unit := range env.Units {
			tableName := fmt.Sprintf("device_%s", strings.ReplaceAll(env.DeviceID, "-", "_"))
			ts := time.UnixMilli(unit.Timestamp).Format("2006-01-02 15:04:05.000")
			payload := string(unit.Payload)

			// 先确保子表存在
			createSQL := fmt.Sprintf(
				"CREATE TABLE IF NOT EXISTS iot_data.%s USING iot_data.device_data TAGS ('%s','%s','%s')",
				tableName, env.DeviceID, env.DomainID, unit.Topic,
			)
			if _, err := w.db.Exec(createSQL); err != nil {
				fmt.Printf("[TDengine] create table error for %s: %v\n", tableName, err)
				continue
			}

			// 插入数据
			insertSQL := fmt.Sprintf(
				"INSERT INTO iot_data.%s VALUES ('%s','%s',0)",
				tableName, ts, payload,
			)
			if _, err := w.db.Exec(insertSQL); err != nil {
				fmt.Printf("[TDengine] insert error for %s: %v\n", tableName, err)
			} else {
				atomic.AddInt64(&w.messagesWritten, 1)
			}
		}
	}
}
