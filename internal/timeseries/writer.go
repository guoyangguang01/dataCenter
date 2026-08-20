package timeseries

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/datacenter/internal/message"
)

type Config struct {
	RESTAddr       string `yaml:"rest_addr"`  // http://localhost:6041
	BatchSize      int    `yaml:"batch_size"`
	FlushInterval  int    `yaml:"flush_interval"`
	BufferCapacity int    `yaml:"buffer_capacity"`
}

type Writer struct {
	config     Config
	httpClient *http.Client
	buffer     chan *message.DeviceEnvelope
	quit       chan struct{}
	wg         sync.WaitGroup
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
	body, _ := json.Marshal(map[string]string{"sql": sql})
	resp, err := w.httpClient.Post(w.config.RESTAddr+"/rest/sql", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return nil
}

func (w *Writer) initDB() error {
	w.execSQL("CREATE DATABASE IF NOT EXISTS iot_data KEEP 90")
	w.execSQL("CREATE STABLE IF NOT EXISTS iot_data.device_data (ts TIMESTAMP, value NCHAR(512), quality INT) TAGS (device_id NCHAR(64), domain_id NCHAR(64), topic NCHAR(128))")
	fmt.Println("[TDengine] database initialized")
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

func (w *Writer) flush(batch []*message.DeviceEnvelope) {
	for _, env := range batch {
		for _, unit := range env.Units {
			ts := time.UnixMilli(unit.Timestamp).Format("2006-01-02 15:04:05.000")
			payload := string(unit.Payload)
			sql := fmt.Sprintf(
				"INSERT INTO iot_data.device_%s USING iot_data.device_data TAGS ('%s','%s','%s') VALUES ('%s','%s',0)",
				env.DeviceID, env.DeviceID, env.DomainID, unit.Topic, ts, payload,
			)
			if err := w.execSQL(sql); err != nil {
				fmt.Println("[TDengine] insert error:", err)
			}
		}
	}
}
