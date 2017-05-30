package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/wlibo666/common-lib/log"
	"github.com/wlibo666/filesync/server/config"
	"github.com/wlibo666/filesync/server/handle"
)

var (
	ConfFile = flag.String("conf", "./conf/server.json", "config file,eg: -conf ./conf/server.json")
)

func RegistryCtlCSignal() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func(c chan os.Signal) {
		sig := <-c
		log.Logger.Info("recv signal:%s then exit", sig.String())
		os.Exit(2)
	}(c)
}

func main() {
	flag.Parse()
	Prepare()
	RegistryCtlCSignal()
	log.Logger.Info("program [%s] start...", os.Args[0])

	handle.StartHeartBeatListener()
	handle.MoniFilesAndSync()
	log.Logger.Info("program [%s] exit...", os.Args[0])
}

func Prepare() error {
	err := config.LoadConfig(*ConfFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "LoadConfig [%s] failed,err:%s\n", *ConfFile, err.Error())
		os.Exit(1)
	}
	log.SetFileLogger(config.GServerConf.LogFile, config.GServerConf.LogFileNum)
	if config.GServerConf.DebugFlag {
		log.SetLoggerDebug()
	}
	return nil
}
