package timeseries

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/taosdata/driver-go/v3/taosRestful"
)

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
