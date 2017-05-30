package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/wlibo666/filesync/lib/common"
)

type FileSyncMoniConf struct {
	DirName   string   `json:"dir"`
	WhiteList []string `json:"white_list"`
}

type FileSyncServerConf struct {
	ListenAddr string              `json:"listen"`
	DebugFlag  bool                `json:"debug"`
	LogFile    string              `json:"log_file"`
	LogFileNum int                 `json:"log_file_num"`
	MoniDirs   []*FileSyncMoniConf `json:"moni_dir"`
}

var (
	GServerConf = &FileSyncServerConf{}
)

func LoadConfig(filename string) error {
	return common.LoadConf(filename, GServerConf)
}

func IsInWhiteList(ipAddr string) bool {
	for _, dir := range GServerConf.MoniDirs {
		for _, addr := range dir.WhiteList {
			if ipAddr == strings.Split(addr, ":")[0] {
				return true
			}
		}
	}
	return false
}

func PrintServerConf(config *FileSyncServerConf) {
	fmt.Fprintf(os.Stdout, "listen:%s\n", config.ListenAddr)
	fmt.Fprintf(os.Stdout, "debug:%v\n", config.DebugFlag)
	fmt.Fprintf(os.Stdout, "log_file:%s\n", config.LogFile)
	fmt.Fprintf(os.Stdout, "log_file_num:%d\n", config.LogFileNum)

	for _, moni := range config.MoniDirs {
		fmt.Fprintf(os.Stdout, "  dir:%s\n", moni.DirName)
		fmt.Fprintf(os.Stdout, "  white_list:%v\n", moni.WhiteList)
	}
}
