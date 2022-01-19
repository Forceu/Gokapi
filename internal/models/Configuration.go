package models

// Configuration is a struct that contains the global configuration
type Configuration struct {
	Authentication   AuthenticationConfig      `json:"Authentication"`
	Port             string                    `json:"Port"`
	ServerUrl        string                    `json:"ServerUrl"`
	RedirectUrl      string                    `json:"RedirectUrl"`
	ConfigVersion    int                       `json:"ConfigVersion"`
	LengthId         int                       `json:"LengthId"`
	DataDir          string                    `json:"DataDir"`
	MaxMemory        int                       `json:"MaxMemory"`
	UseSsl           bool                      `json:"UseSsl"`
	MaxFileSizeMB    int                       `json:"MaxFileSizeMB"`
	Sessions         map[string]Session        `json:"Sessions"`
	Files            map[string]File           `json:"Files"`
	Hotlinks         map[string]Hotlink        `json:"Hotlinks"`
	DownloadStatus   map[string]DownloadStatus `json:"DownloadStatus"`
	ApiKeys          map[string]ApiKey         `json:"ApiKeys"`
	DefaultDownloads int
	DefaultExpiry    int
	DefaultPassword  string
}
