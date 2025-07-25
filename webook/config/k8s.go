//go:build k8s

// 使用 k8s 标签
package config

var Config = config{
	DB: DBConfig{
		// 本地连接
		DSN: "root:root@tcp(webook-mysql:11308)/webook",
	},
	Redis: RedisConfig{
		Addr: "webook-redis:10379",
	},
}
