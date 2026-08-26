package config

type config struct {
	DB    DBConfig
	Redis RedisConfig
	File  FileConfig
}

type DBConfig struct {
	DSN string
}

type RedisConfig struct {
	Addr string
}

type FileConfig struct {
	RootPath  string
	ChunkPath string
}
