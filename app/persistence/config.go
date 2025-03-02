package persistence

import "sync"

var (
	mu     sync.Mutex
	config = struct {
		dir        string
		dbFilename string
	}{
		dir:        ".",
		dbFilename: "dump.rdb",
	}
)

func InitConfig(dir, filename string) {
	mu.Lock()
	defer mu.Unlock()
	config.dir = dir
	config.dbFilename = filename
}

func GetConfig() (string, string) {
	mu.Lock()
	defer mu.Unlock()
	return config.dir, config.dbFilename
}
