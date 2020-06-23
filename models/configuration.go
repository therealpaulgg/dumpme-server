package models

// Configuration holds all user configuration parameters stored in the config.json file.
type Configuration struct {
	StorageType string `json:"storagetype"`
	StoragePath string `json:"storagePath"`
	Port        int    `json:"port"`
	Environment string `json:"environment"`
	FileLimit   string `json:"fileLimit"`
}

// AppConfiguration is the configuration that the application uses during runtime.
type AppConfiguration struct {
	FileLimit int64
}
