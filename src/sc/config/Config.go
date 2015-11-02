
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

type ConfigBuildings struct {
	PoolSize uint16 `json:"pool_size"`
}

type Config struct {
    Daemonize bool				`json:"daemonize"`
    PidFilepath string			`json:"pidfile"`
	Cassandra ConfigCassandra	`json:"cassandra"`
	Http ConfigHTTP				`json:"http"`
	Logger ConfigLogger			`json:"logger"`
	Auth ConfigAuth				`json:"auth"`
	Buildings ConfigBuildings	`json:"buildings"`
}
