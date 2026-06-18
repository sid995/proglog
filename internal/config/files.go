package config

import (
	"os"
	"path/filepath"
)

var (
	CAFile         = configFile("ca.pem")
	ServerCertFile = configFile("server.pem")
	ServerKeyFile  = configFile("server-key.pem")
	ClientCertFile = configFile("client.pem")
	ClientKeyFile  = configFile("client-key.pem")
)

func configFile(filename string) string {
	if dir := os.Getenv("CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, filename)
	}
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	for {
		configDir := filepath.Join(dir, ".proglog")
		if stat, err := os.Stat(configDir); err == nil && stat.IsDir() {
			return filepath.Join(configDir, filename)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return filepath.Join(configDir, filename)
		}
		dir = parent
	}
}
