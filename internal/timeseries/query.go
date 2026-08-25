package timeseries

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/taosdata/driver-go/v3/taosRestful"
)

// MetricLatestItem 单个设备的指标最新值
type MetricLatestItem struct {
	DeviceID string  `json:"device_id"`
	TS       string  `json:"ts"`
	Value    float64 `json:"value"`
	Quality  int     `json:"quality"`
}

// MetricLatestResponse 跨设备指标最新值响应
type MetricLatestResponse struct {
	Metric string             `json:"metric"`
	Devices []MetricLatestItem `json:"devices"`
}

// MetricDeviceHistory 单个设备的指标历史数据
type MetricDeviceHistory struct {
	DeviceID string    `json:"device_id"`
	Data     [][]interface{} `json:"data"` // [[ts, value], ...]
}

// MetricHistoryResponse 跨设备指标历史响应
type MetricHistoryResponse struct {
	Metric  string               `json:"metric"`
	Devices []MetricDeviceHistory `json:"devices"`
}

type QueryService struct {
	db *sql.DB
}

func NewQueryService(config Config) (*QueryService, error) {
	db, err := sql.Open("taosRestful", config.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open TDengine: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping TDengine: %w", err)
	}
	return &QueryService{db: db}, nil
}

func (s *QueryService) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

type QueryResult struct {
	ColumnNames  []string        `json:"column_names"`
	ColumnTypes  []string        `json:"column_types"`
	Rows         [][]interface{} `json:"rows"`
	RowsAffected int64           `json:"rows_affected"`
}

// Query 执行 SQL 查询
func (s *QueryService) Query(sqlStr string) (*QueryResult, error) {
	rows, err := s.db.Query(sqlStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	colTypes, _ := rows.ColumnTypes()
	typeNames := make([]string, len(colTypes))
	for i, ct := range colTypes {
		typeNames[i] = ct.DatabaseTypeName()
	}

	result := &QueryResult{
		ColumnNames: columns,
		ColumnTypes: typeNames,
		Rows:        make([][]interface{}, 0),
	}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		ptrs := make([]interface{}, len(columns))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		result.Rows = append(result.Rows, values)
	}

	return result, rows.Err()
}

// Exec 执行写入 SQL
func (s *QueryService) Exec(sqlStr string) (int64, error) {
	res, err := s.db.Exec(sqlStr)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// QueryByDevice 按设备查询数据
func (s *QueryService) QueryByDevice(deviceID string, start, end time.Time) (*QueryResult, error) {
	sqlStr := fmt.Sprintf(
		"SELECT ts, val, quality FROM iot_data.device_%s WHERE ts >= '%s' AND ts <= '%s'",
		strings.ReplaceAll(deviceID, "-", "_"),
		start.Format("2006-01-02 15:04:05"),
		end.Format("2006-01-02 15:04:05"),
	)
	return s.Query(sqlStr)
}

// QueryByDeviceAndTopic 按设备和主题查询
func (s *QueryService) QueryByDeviceAndTopic(deviceID, topic string, start, end time.Time) (*QueryResult, error) {
	sqlStr := fmt.Sprintf(
		"SELECT ts, val, quality FROM iot_data.device_%s WHERE topic_name = '%s' AND ts >= '%s' AND ts <= '%s'",
		strings.ReplaceAll(deviceID, "-", "_"), topic,
		start.Format("2006-01-02 15:04:05"),
		end.Format("2006-01-02 15:04:05"),
	)
	return s.Query(sqlStr)
}

// QueryLatest 查询设备最新数据
func (s *QueryService) QueryLatest(deviceID string, limit int) (*QueryResult, error) {
	sqlStr := fmt.Sprintf(
		"SELECT LAST_ROW(ts, val) FROM iot_data.device_%s",
		strings.ReplaceAll(deviceID, "-", "_"),
	)
	return s.Query(sqlStr)
}

// QueryAggregation 聚合查询
func (s *QueryService) QueryAggregation(deviceID, topic, function string, start, end time.Time, interval string) (*QueryResult, error) {
	sqlStr := fmt.Sprintf(
		"SELECT _wstart, %s(val) FROM iot_data.device_%s WHERE topic_name = '%s' AND ts >= '%s' AND ts <= '%s' INTERVAL(%s)",
		function, strings.ReplaceAll(deviceID, "-", "_"), topic,
		start.Format("2006-01-02 15:04:05"),
		end.Format("2006-01-02 15:04:05"),
		interval,
	)
	return s.Query(sqlStr)
}

// QueryLatestAll 查询所有设备最新数据（使用 stable 的 GROUP BY）
// domainID 非空时按域过滤
func (s *QueryService) QueryLatestAll(domainID string) (*QueryResult, error) {
	sqlStr := "SELECT device_id, LAST_ROW(ts), LAST_ROW(val), LAST_ROW(quality) FROM iot_data.device_data"
	if domainID != "" {
		sqlStr += fmt.Sprintf(" WHERE domain_id = '%s'", domainID)
	}
	sqlStr += " GROUP BY device_id"
	return s.Query(sqlStr)
}

// parseMetricFromVal 从 val JSON 中提取指定指标的数值
func parseMetricFromVal(valStr string, metricKey string) (float64, bool) {
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(valStr), &parsed); err != nil {
		return 0, false
	}
	v, ok := parsed[metricKey]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return n, true
	case json.Number:
		f, err := n.Float64()
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}

// QueryLatestMetric 查询所有设备中指定指标的最新值
func (s *QueryService) QueryLatestMetric(metricKey, domainID string) (*MetricLatestResponse, error) {
	result, err := s.QueryLatestAll(domainID)
	if err != nil {
		return &MetricLatestResponse{Metric: metricKey}, nil
	}

	resp := &MetricLatestResponse{Metric: metricKey, Devices: make([]MetricLatestItem, 0)}
	for _, row := range result.Rows {
		if len(row) < 4 {
			continue
		}
		deviceID, _ := row[0].(string)
		ts := fmt.Sprintf("%v", row[1])
		valStr := fmt.Sprintf("%v", row[2])
		quality := 0
		if q, ok := row[3].(int64); ok {
			quality = int(q)
		}

		value, ok := parseMetricFromVal(valStr, metricKey)
		if !ok {
			continue
		}

		// 还原 device_id（TDengine 下划线替换）
		realDeviceID := strings.ReplaceAll(deviceID, "_", "-")
		resp.Devices = append(resp.Devices, MetricLatestItem{
			DeviceID: realDeviceID,
			TS:       ts,
			Value:    value,
			Quality:  quality,
		})
	}
	return resp, nil
}

// QueryMetricHistory 查询各设备指定指标的历史数据
func (s *QueryService) QueryMetricHistory(metricKey, domainID string, start, end time.Time) (*MetricHistoryResponse, error) {
	// 先获取所有设备列表
	allResult, err := s.QueryLatestAll(domainID)
	if err != nil {
		return &MetricHistoryResponse{Metric: metricKey}, nil
	}

	resp := &MetricHistoryResponse{Metric: metricKey, Devices: make([]MetricDeviceHistory, 0)}

	for _, row := range allResult.Rows {
		if len(row) < 1 {
			continue
		}
		deviceID, _ := row[0].(string)
		realDeviceID := strings.ReplaceAll(deviceID, "_", "-")

		// 查询该设备历史数据
		sqlStr := fmt.Sprintf(
			"SELECT ts, val, quality FROM iot_data.device_%s WHERE ts >= '%s' AND ts <= '%s' ORDER BY ts ASC",
			deviceID,
			start.Format("2006-01-02 15:04:05"),
			end.Format("2006-01-02 15:04:05"),
		)
		historyResult, err := s.Query(sqlStr)
		if err != nil {
			continue
		}

		deviceData := MetricDeviceHistory{DeviceID: realDeviceID, Data: make([][]interface{}, 0)}
		for _, hRow := range historyResult.Rows {
			if len(hRow) < 2 {
				continue
			}
			ts := fmt.Sprintf("%v", hRow[0])
			valStr := fmt.Sprintf("%v", hRow[1])

			value, ok := parseMetricFromVal(valStr, metricKey)
			if !ok {
				continue
			}
			deviceData.Data = append(deviceData.Data, []interface{}{ts, value})
		}

		if len(deviceData.Data) > 0 {
			resp.Devices = append(resp.Devices, deviceData)
		}
	}
	return resp, nil
}
