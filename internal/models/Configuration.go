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
	Files            map[string]File           `json:"Files"`
	DefaultDownloads int
	DefaultExpiry    int
	DefaultPassword  string
}

// migrate: hotlinks, apikeys
