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
	return s.adapter.GetServers(ctx)
}

func (s *Service) RunSpeedtest(ctx context.Context) (string, error) {

	req, err := s.adapter.RunSpeedtest(ctx)
	if err != nil {
		return "", err
	}

	res := s.metrics(req)

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

	b.WriteString("# HELP qms_speedtest_upload_mbps Upload speed in Mbps\n")
	b.WriteString("# TYPE qms_speedtest_upload_mbps gauge\n")
	b.WriteString(fmt.Sprintf("qms_speedtest_upload_mbps %.6f\n", res.Upload))

	// Несколько label-меток с информацией (как отдельный gauge со значением 1)
	labels := fmt.Sprintf("server=\"%s\",city=\"%s\",region=\"%s\",ip=\"%s\",isp=\"%s\"",
		escape(res.Server), escape(res.City), escape(res.RegionName), escape(res.IP), escape(res.ISP))
	b.WriteString("# HELP qms_speedtest_info Meta information\n")
	b.WriteString("# TYPE qms_speedtest_info gauge\n")
	b.WriteString(fmt.Sprintf("qms_speedtest_info{%s} 1\n", labels))

	return b.String()
}

func (s *Service) getLabels(res *entities.SpeedtestResult) string {
	return fmt.Sprintf(`datetime="%s",server="%s",city="%s",region_name="%s",ip="%s",isp="%s",ping="%d",jitter="%d",data="%s"`,
		res.DateTime, res.Server, res.City, res.RegionName, res.IP, res.ISP, res.Ping, res.Jitter, res.Data)
}

func (s *Service) metrics(res *entities.SpeedtestResult) string {
	labels := s.getLabels(res)
	status := 0
	if res.ResultURL != "" {
		status = 1
	}

	return fmt.Sprintf(`# HELP speedtest_download_speed_Bps Download speed
# TYPE speedtest_download_speed_Bps gauge
speedtest_download_speed_Bps{%s} %f
# HELP speedtest_upload_speed_Bps Upload speed
# TYPE speedtest_upload_speed_Bps gauge  
speedtest_upload_speed_Bps{%s} %f
# HELP speedtest_up Status
# TYPE speedtest_up gauge
speedtest_up{result="%s"} %d`,
		labels, res.Download,
		labels, res.Upload,
		res.ResultURL, status)
}

func escape(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}
