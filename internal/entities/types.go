package entities

// Server описывает сервер speedtest.
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

// PingStats статистика пинга для аплоада/даунлоада.
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

// SpeedtestResult результат теста скорости, совместим с JSON от qms_lib.
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
	Data         string    `json:"data"`
	ResultURL    string    `json:"result"`
}
