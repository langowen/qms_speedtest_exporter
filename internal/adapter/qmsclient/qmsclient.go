package qmsclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/langowen/qms_speedtest_exporter/internal/config"
	"github.com/langowen/qms_speedtest_exporter/internal/entities"
)

// Client — адаптер для работы с qms_lib.
type Client struct {
	log *slog.Logger
	cfg *config.Config
}

func NewQMCClient(log *slog.Logger, cfg *config.Config) *Client {
	return &Client{log: log, cfg: cfg}
}

// GetServers запускает бинарь для получения server_data и парсит JSON в []Server
func (c *Client) GetServers(ctx context.Context) ([]entities.Server, error) {
	cmd := exec.CommandContext(ctx, c.cfg.BinaryPath, "-L")

	start := time.Now()
	if err := cmd.Run(); err != nil {
		c.log.Error("Failed to get servers", "error", err)
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, entities.ErrTimeout
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			return nil, entities.ErrCancelled
		}
		return nil, fmt.Errorf("%w: %v", entities.ErrExecFailed, err)
	}

	if err := c.waitForFile(ctx, c.cfg.ServerDataPath); err != nil {
		return nil, fmt.Errorf("server_data not ready: %w", err)
	}

	c.log.Debug("servers list ready", slog.Duration("duration", time.Since(start)))

	b, err := os.ReadFile(c.cfg.ServerDataPath)
	if err != nil {
		return nil, fmt.Errorf("read server_data: %w", err)
	}
	var servers []entities.Server
	if err := json.Unmarshal(b, &servers); err != nil {
		return nil, fmt.Errorf("parse server_data: %w", err)
	}
	return servers, nil
}

// RunSpeedtest запускает speedtest и парсит результат
func (c *Client) RunSpeedtest(ctx context.Context) (*entities.SpeedtestResult, error) {
	if err := os.MkdirAll(filepath.Dir(c.cfg.TestResultPath), 0755); err != nil {
		return nil, fmt.Errorf("can't create result test directory: %w", err)
	}

	args := []string{"-O", c.cfg.TestResultPath, "-F", "json"}

	if c.cfg.ServerID != 0 {
		args = append([]string{"-S", strconv.Itoa(c.cfg.ServerID)}, args...)
	}

	start := time.Now()
	cmd := exec.CommandContext(ctx, c.cfg.BinaryPath, args...)
	if err := cmd.Run(); err != nil {
		select {
		case <-ctx.Done():
			return nil, entities.ErrTimeout
		default:
			// Проверяем, является ли ошибка "signal: aborted"
			if isAbortError(err) {
				c.log.Debug("Speedtest aborted (expected in non-TTY environment)", slog.Any("err", err))
				// Игнорируем только эту конкретную ошибку
			} else {
				c.log.Error("speedtest run returned error", slog.Any("err", err))
				return nil, fmt.Errorf("speedtest failed: %v", err)
			}
		}

	}

	c.log.Debug("speedtest finished", slog.Duration("duration", time.Since(start)))

	b, err := os.ReadFile(c.cfg.TestResultPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", entities.ErrResultMissing, err)
	}
	var res entities.SpeedtestResult
	if err := json.Unmarshal(b, &res); err != nil {
		c.log.Error("parse speedtest result", slog.Any("err", err))
		return nil, fmt.Errorf("parse test result: %w", err)
	}
	res.Duration = time.Since(start)
	return &res, nil
}

func (c *Client) waitForFile(ctx context.Context, filename string) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, 40*time.Second)
	defer cancel()
	for {
		select {
		case <-ctxTimeout.Done():
			return fmt.Errorf("timeout waiting for file %s", filename)
		default:
			if st, err := os.Stat(filename); err == nil && st.Size() > 0 {
				c.log.Debug("file ready", slog.String("file", filename))
				return nil
			}
			i := 0
			i++
			sleepTime := time.Duration(i) * 100 * time.Millisecond
			time.Sleep(sleepTime * time.Millisecond)
		}
	}
}

func isAbortError(err error) bool {
	if err == nil {
		return false
	}

	errorStr := err.Error()

	// Варианты, которые могут быть у "signal: aborted"
	abortPatterns := []string{
		"signal: aborted",
		"signal: abort",
		"exit status 134",
		"SIGABRT",
	}

	for _, pattern := range abortPatterns {
		if strings.Contains(errorStr, pattern) {
			return true
		}
	}

	return false
}
