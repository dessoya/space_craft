
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

type ConfigCassandra struct {
	IP string `json:"ip"`
}

type Config struct {
	Cassandra ConfigCassandra	`json:"cassandra"`
	Http ConfigHTTP				`json:"http"`
	Logger ConfigLogger			`json:"logger"`
	Auth ConfigAuth				`json:"auth"`
}
