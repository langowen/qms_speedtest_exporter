package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var (
	patchBin        = "bin/qms_lib"
	patchServerData = "server_data"
	patchTestRes    = "data/test.json"
	patchServersRes = "data/servers.json"
)

type Server struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	City        string `json:"city"`
	Lat         string `json:"lat"`
	Lng         string `json:"lng"`
	Src         string `json:"src"`
	Source      string `json:"source"`
	Port        int    `json:"port"`
	RegionName  string `json:"region_name"`
	RegionOkato string `json:"region_okato"`
	ExternalID  string `json:"external_id"`
	Distance    int    `json:"distance"`
}

type SpeedtestResult struct {
	DateTime     string    `json:"datetime"`
	Server       string    `json:"server"`
	City         string    `json:"city"`
	RegionName   string    `json:"region_name"`
	IP           string    `json:"ip"`
	ISP          string    `json:"isp"`
	Ping         int       `json:"ping"`
	Jitter       int       `json:"jitter"`
	Download     float64   `json:"download"`
	DownloadPing PingStats `json:"download_ping"`
	Upload       float64   `json:"upload"`
	UploadPing   PingStats `json:"upload_ping"`
	Data         float64   `json:"data"`
	ResultURL    string    `json:"result"`
}

type PingStats struct {
	Count  int `json:"count"`
	Min    int `json:"min"`
	Max    int `json:"max"`
	Mean   int `json:"mean"`
	Median int `json:"median"`
	IQR    int `json:"iqr"`
	IQM    int `json:"iqm"`
	Jitter int `json:"jitter"`
}

// Получаем список серверов
func getServers(ctx context.Context) ([]Server, error) {
	// Сначала запускаем бинарник чтобы создать server_data
	cmd := exec.CommandContext(ctx, patchBin, "-L")

	start := time.Now()
	// Запускаем и ждем завершения
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to execute qms_lib: %v", err)
	}

	// Ждем создания файла с проверками
	if err := waitForFile(patchServerData, ctx); err != nil {
		return nil, fmt.Errorf("server_data file was not created: %v", err)
	}

	fmt.Printf("Duration get speedtest servers list: %s\n", time.Since(start))

	content, err := os.ReadFile(patchServerData)
	if err != nil {
		return nil, fmt.Errorf("failed to read server_data file: %v", err)
	}

	var servers []Server
	if err := json.Unmarshal(content, &servers); err != nil {
		return nil, fmt.Errorf("failed to parse server_data JSON: %v", err)
	}

	return servers, nil
}

// waitForFile ждет пока файл не будет создан
func waitForFile(filename string, ctx context.Context) error {
	start := time.Now()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for file %s", filename)
		default:
			if _, err := os.Stat(filename); err == nil {
				// Файл существует, проверяем что он не пустой
				if info, err := os.Stat(filename); err == nil && info.Size() > 0 {
					fmt.Printf("file %s exists, durations %s\n", filename, time.Since(start))
					return nil
				}
			}
		}
	}
}

// Сохраняем список серверов в наш JSON формат
func saveServersToFile(servers []Server, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(servers)
}

// Запускаем speedtest на конкретном сервере
func runSpeedtest(ctx context.Context, serverID int) (*SpeedtestResult, error) {
	args := []string{"-O", patchTestRes, "-F", "json"}
	if serverID != 0 {
		args = append([]string{"-S", strconv.Itoa(serverID)}, args...)
	}

	start := time.Now()
	cmd := exec.CommandContext(ctx, patchBin, args...)

	if err := cmd.Run(); err != nil {
		// Проверяем, не была ли ошибка из-за отмены контекста
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("speedtest cancelled: %v", ctx.Err())
		default:
			// Проверяем, является ли ошибка "signal: aborted"
			if isAbortError(err) {
				fmt.Printf("⚠️ Speedtest aborted (expected in non-TTY environment)\n")
				// Игнорируем только эту конкретную ошибку
			} else {
				// Все другие ошибки считаем критическими
				return nil, fmt.Errorf("speedtest failed: %v", err)
			}
		}
	}

	fmt.Printf("Duration speedtest resoult: %s\n", time.Since(start))

	// Читаем результат
	content, err := os.ReadFile(patchTestRes)
	if err != nil {
		return nil, fmt.Errorf("failed to read results: %v", err)
	}

	// Парсим JSON
	var result SpeedtestResult
	if err := json.Unmarshal(content, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	return &result, nil
}

// Выводим список серверов
func printServers(servers []Server) {
	fmt.Println("\n" + "🌐" + " AVAILABLE SERVERS " + "🌐")
	fmt.Println("═" + "══════════════════════════════════════════════════")
	fmt.Printf("%-10s %-20s %-15s %-15s %s\n", "ID", "Name", "City", "Provider", "Distance")
	fmt.Println("─" + "──────────────────────────────────────────────────")

	for _, server := range servers {
		fmt.Printf("%-10d %-20s %-15s %-15s %d km\n",
			server.ID,
			truncateString(server.Name, 180),
			truncateString(server.City, 130),
			truncateString(server.Source, 130),
			server.Distance)
	}
	fmt.Println("─" + "──────────────────────────────────────────────────")
	fmt.Printf("Total servers: %d\n", len(servers))
}

// Обрезаем строку если слишком длинная
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	fmt.Println("🚀 QMS Speedtest Manager")

	// Создаем директорию для данных
	if err := os.MkdirAll("data", 0755); err != nil {
		fmt.Printf("Не удалось создать директорию %s", err)
	}

	// Получаем список серверов из server_data
	fmt.Println("\n📡 Reading servers list from server_data...")
	servers, err := getServers(ctx)
	if err != nil {
		fmt.Printf("❌ Error reading servers: %v\n", err)
		return
	}

	// Сохраняем в наш JSON формат
	err = saveServersToFile(servers, patchServersRes)
	if err != nil {
		fmt.Printf("❌ Error saving servers: %v\n", err)
		return
	}
	fmt.Printf("✅ Saved %d servers to data/servers.json\n", len(servers))

	// Выводим список серверов
	printServers(servers)

	// Запускаем тест на автоматически выбранном сервере (ближайшем)
	fmt.Println("\n🧪 Running speedtest...")
	serverID := 0

	result, err := runSpeedtest(ctx, serverID)
	if err != nil {
		fmt.Printf("❌ Speedtest failed: %v\n", err)
		return
	}

	// Выводим результаты
	printResults(result)
}

func printResults(result *SpeedtestResult) {
	fmt.Println("\n" + "📊" + " SPEEDTEST RESULTS " + "📊")
	fmt.Println("═" + "══════════════════════════════")
	fmt.Printf("🕒 %s\n", formatTime(result.DateTime))
	fmt.Printf("🌐 %s (%s)\n", result.ISP, result.IP)
	fmt.Printf("📍 %s, %s\n", result.City, result.RegionName)
	fmt.Printf("🖥️  Server: %s\n", result.Server)
	fmt.Println("─" + "──────────────────────────────")
	fmt.Printf("📥 Download:   %8.2f Mbit/s\n", result.Download)
	fmt.Printf("📤 Upload:     %8.2f Mbit/s\n", result.Upload)
	fmt.Printf("⏱️  Ping:       %8d ms\n", result.Ping)
	fmt.Printf("📊 Jitter:     %8d ms\n", result.Jitter)
	fmt.Printf("💾 Data used:  %8.2f MB\n", result.Data)
	fmt.Println("─" + "──────────────────────────────")
	fmt.Printf("🔗 Result: %s\n", result.ResultURL)
}

func formatTime(datetime string) string {
	t, err := time.Parse(time.RFC3339, datetime)
	if err != nil {
		return datetime
	}
	return t.Format("2006-01-02 15:04:05")
}

// isAbortError проверяет, является ли ошибка "signal: aborted"
func isAbortError(err error) bool {
	if err == nil {
		return false
	}

	// Проверяем разные варианты представления ошибки
	errorStr := err.Error()

	// Варианты, которые могут быть у "signal: aborted"
	abortPatterns := []string{
		"signal: aborted",
		"signal: abort",
		"exit status 134", // Код выхода для SIGABRT
		"SIGABRT",
	}

	for _, pattern := range abortPatterns {
		if strings.Contains(errorStr, pattern) {
			return true
		}
	}

	return false
}
