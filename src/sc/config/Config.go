
package config

type ConfigHTTP struct {
	Port uint16 `json:"port"`
}

type ConfigLogger struct {
	Path string `json:"path"`
}

type ConfigAuth struct {
	Methods []string `json:"methods"`
}

type Config struct {
	Http ConfigHTTP			`json:"http"`
	Logger ConfigLogger		`json:"logger"`
	Auth ConfigAuth			`json:"auth"`
}
