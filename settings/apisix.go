package settings

type APISIX struct {
	BaseURL     string `json:"base_url" env:"BASE_URL" binding:"omitempty,url"`
	APIKey      string `json:"api_key" env:"API_KEY" binding:"omitempty,safety_text"`
	ReplacePath string `json:"replace_path" env:"REPLACE_PATH" binding:"omitempty,safety_text"`
}

var APISIXSettings = &APISIX{}
