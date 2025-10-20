package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/langowen/qms_speedtest_exporter/internal/config"
	"github.com/langowen/qms_speedtest_exporter/internal/entities"
)

// SpeedtestAdapter интерфейс для адаптера qmsclient.
type SpeedtestAdapter interface {
	GetServers(ctx context.Context) ([]entities.Server, error)
	RunSpeedtest(ctx context.Context) (*entities.SpeedtestResult, error)
	RemoveResult(path string)
}

// Service бизнес-логика поверх адаптера.
type Service struct {
	adapter SpeedtestAdapter
	cfg     *config.Config
}

func NewService(adapter SpeedtestAdapter, cfg *config.Config) *Service {
	return &Service{adapter: adapter, cfg: cfg}
}

func (s *Service) GetServers(ctx context.Context) ([]entities.Server, error) {
	res, err := s.adapter.GetServers(ctx)
	if err != nil {
		return nil, err
	}

	go func() {
		s.adapter.RemoveResult(s.cfg.ServerDataPath)
	}()

	return res, nil
}

func (s *Service) RunSpeedtest(ctx context.Context) (string, error) {

	req, err := s.adapter.RunSpeedtest(ctx)
	if err != nil {
		return "", err
	}

	res := s.ToPrometheusMetrics(req)

	go func() {
		s.adapter.RemoveResult(s.cfg.TestResultPath)
	}()

	return res, nil
}

// ToPrometheusMetrics преобразует результат speedtest в формат Prometheus exposition (text/plain; version=0.0.4)
func (s *Service) ToPrometheusMetrics(res *entities.SpeedtestResult) string {
	// Простые gauge-метрики без labels для минимальной совместимости
	var b strings.Builder
	// помощь по метрикам
	b.WriteString("# HELP qms_speedtest_ping_ms Average ping in ms\n")
	b.WriteString("# TYPE qms_speedtest_ping_ms gauge\n")
	b.WriteString(fmt.Sprintf("qms_speedtest_ping_ms %d\n", res.Ping))

	b.WriteString("# HELP qms_speedtest_jitter_ms Jitter in ms\n")
	b.WriteString("# TYPE qms_speedtest_jitter_ms gauge\n")
	b.WriteString(fmt.Sprintf("qms_speedtest_jitter_ms %d\n", res.Jitter))

	b.WriteString("# HELP qms_speedtest_download_mbps Download speed in Mbps\n")
	b.WriteString("# TYPE qms_speedtest_download_mbps gauge\n")
	b.WriteString(fmt.Sprintf("qms_speedtest_download_mbps %.6f\n", res.Download))

	b.WriteString("# HELP qms_speedtest_download_ping Download ping stats\n")
	b.WriteString("# TYPE qms_speedtest_download_ping gauge\n")
	pingStatDown := s.getLabelsPing(&res.DownloadPing)
	b.WriteString(fmt.Sprintf("qms_speedtest_download_ping{%s} 1\n", pingStatDown))

	b.WriteString("# HELP qms_speedtest_upload_mbps Upload speed in Mbps\n")
	b.WriteString("# TYPE qms_speedtest_upload_mbps gauge\n")
	b.WriteString(fmt.Sprintf("qms_speedtest_upload_mbps %.6f\n", res.Upload))

	b.WriteString("# HELP qms_speedtest_upload_ping Upload ping stats\n")
	b.WriteString("# TYPE qms_speedtest_upload_ping gauge\n")
	pingStatUpload := s.getLabelsPing(&res.UploadPing)
	b.WriteString(fmt.Sprintf("qms_speedtest_upload_ping{%s} 1\n", pingStatUpload))

	labels := fmt.Sprintf("datetime=\"%s\",server=\"%s\",city=\"%s\",region=\"%s\",ip=\"%s\",isp=\"%s\",data=\"%f\"",
		escape(res.DateTime), escape(res.Server), escape(res.City), escape(res.RegionName), escape(res.IP), escape(res.ISP), res.Data)
	b.WriteString("# HELP qms_speedtest_info Meta information\n")
	b.WriteString("# TYPE qms_speedtest_info gauge\n")
	b.WriteString(fmt.Sprintf("qms_speedtest_info{%s} 1\n", labels))

	b.WriteString("# HELP qms_speedtest_scrape_duration_seconds Duration speedtest in seconds\n")
	b.WriteString("# TYPE qms_speedtest_scrape_duration_seconds gauge\n")
	b.WriteString(fmt.Sprintf("qms_speedtest_scrape_duration_seconds %.2f\n", res.Duration.Seconds()))

	return b.String()
}

func (s *Service) getLabelsPing(res *entities.PingStats) string {
	labels := fmt.Sprintf("count=\"%d\",min=\"%d\",max=\"%d\",mean=\"%d\",median=\"%d\",iqr=\"%d\",iqm=\"%d\",jitter=\"%d\"",
		res.Count, res.Min, res.Max, res.Mean, res.Median, res.IQR, res.IQM, res.Jitter)

	return labels
}

func escape(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}
