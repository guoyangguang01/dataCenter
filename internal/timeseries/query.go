package timeseries

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type QueryService struct {
	config     Config
	httpClient *http.Client
}

func NewQueryService(config Config) *QueryService {
	return &QueryService{
		config:     config,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

type QueryResult struct {
	ColumnNames []string        `json:"column_names"`
	ColumnTypes []string        `json:"column_types"`
	Rows        [][]interface{} `json:"rows"`
	RowsAffected int64          `json:"rows_affected"`
}

// Query 执行 SQL 查询
func (s *QueryService) Query(sql string) (*QueryResult, error) {
	body, _ := json.Marshal(map[string]string{"sql": sql})
	resp, err := s.httpClient.Post(s.config.RESTAddr+"/rest/sql", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result QueryResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &result, nil
}

// QueryByDevice 按设备查询数据
func (s *QueryService) QueryByDevice(deviceID string, start, end time.Time) (*QueryResult, error) {
	sql := fmt.Sprintf(
		"SELECT ts, value, quality FROM iot_data.device_%s WHERE ts >= '%s' AND ts <= '%s'",
		deviceID,
		start.Format("2006-01-02 15:04:05"),
		end.Format("2006-01-02 15:04:05"),
	)
	return s.Query(sql)
}

// QueryByDeviceAndTopic 按设备和主题查询
func (s *QueryService) QueryByDeviceAndTopic(deviceID, topic string, start, end time.Time) (*QueryResult, error) {
	sql := fmt.Sprintf(
		"SELECT ts, value, quality FROM iot_data.device_%s WHERE topic = '%s' AND ts >= '%s' AND ts <= '%s'",
		deviceID, topic,
		start.Format("2006-01-02 15:04:05"),
		end.Format("2006-01-02 15:04:05"),
	)
	return s.Query(sql)
}

// QueryLatest 查询设备最新数据
func (s *QueryService) QueryLatest(deviceID string, limit int) (*QueryResult, error) {
	sql := fmt.Sprintf(
		"SELECT LAST_ROW(ts, value) FROM iot_data.device_%s",
		deviceID,
	)
	return s.Query(sql)
}

// QueryAggregation 聚合查询
func (s *QueryService) QueryAggregation(deviceID, topic, function string, start, end time.Time, interval string) (*QueryResult, error) {
	sql := fmt.Sprintf(
		"SELECT _wstart, %s(value) FROM iot_data.device_%s WHERE topic = '%s' AND ts >= '%s' AND ts <= '%s' INTERVAL(%s)",
		function, deviceID, topic,
		start.Format("2006-01-02 15:04:05"),
		end.Format("2006-01-02 15:04:05"),
		interval,
	)
	return s.Query(sql)
}
