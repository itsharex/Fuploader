package types

type AppVersion struct {
	Version      string `json:"version"`
	BuildTime    string `json:"buildTime"`
	GoVersion    string `json:"goVersion"`
	WailsVersion string `json:"wailsVersion"`
}
