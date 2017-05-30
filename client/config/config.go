package config

import (
	"github.com/wlibo666/filesync/lib/common"
)

type FileSyncConf struct {
	ServerDirName string `json:"server_dir"`
	LocalDirName  string `json:"local_dir"`
	ServerAddr    string `json:"server_addr"`
}

type FileSyncClientConf struct {
	ListenAddr string          `json:"listen"`
	DebugFlag  bool            `json:"debug"`
	LogFile    string          `json:"log_file"`
	LogFileNum int             `json:"log_file_num"`
	SyncDirs   []*FileSyncConf `json:"sync_dir"`
}

var (
	GClientConf = &FileSyncClientConf{}
)

func LoadConfig(filename string) error {
	return common.LoadConf(filename, GClientConf)
}
