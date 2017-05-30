package handle

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/golang/protobuf/proto"
	"github.com/wlibo666/common-lib/log"
	"github.com/wlibo666/filesync/lib/common"
	syncproto "github.com/wlibo666/filesync/lib/proto"
	"github.com/wlibo666/filesync/server/config"
)

var (
	ERR_ONLY_SUPPORT_HEARTBEAT_MSG = errors.New("Only support heartbeat msg in this port")

	eventChan = make(chan fsnotify.Event, syncproto.SYNC_FILE_NUM_ONETIME)

	ClientsAddr   = make(map[string]bool)
	HeartBeatList = make(map[string]int64)
)

func processHeartBeat(conn net.Conn) error {
	maxTry := syncproto.MAX_RETRY_TIME
	tmpTry := 0
	clientAddr := conn.RemoteAddr().String()
	for {
		if tmpTry >= maxTry {
			ClientsAddr[strings.Split(clientAddr, ":")[0]] = false
			log.Logger.Warn("Lost client:%s.", clientAddr)
			return fmt.Errorf("client:%s lost.")
		}
		// read heartbeat request msg
		msgData, err := common.ReadMsg(conn)
		if err != nil {
			log.Logger.Warn("ReadMsg from conn:%s failed,err:%s", clientAddr, err.Error())
			tmpTry++
			time.Sleep(time.Duration(syncproto.HEART_BEAT_INTERVAL) * time.Second)
			continue
		}
		msg := &syncproto.FileSyncProto{}
		err = proto.Unmarshal(msgData, msg)
		if err != nil {
			log.Logger.Warn("proto.Unmarshal failed,err:%s", err.Error())
			tmpTry++
			time.Sleep(time.Duration(syncproto.HEART_BEAT_INTERVAL) * time.Second)
			continue
		}
		syncproto.LogMsg(conn, msg)
		if msg.GetMsgType() != syncproto.PROTO_MSG_HEART_BETA_REQ {
			common.WriteMsg([]byte(ERR_ONLY_SUPPORT_HEARTBEAT_MSG.Error()), conn)
			tmpTry++
			time.Sleep(time.Duration(syncproto.HEART_BEAT_INTERVAL) * time.Second)
			continue
		}

		// write heartbeat response msg
		msgRes := &syncproto.FileSyncProto{
			Version:    proto.Uint32(syncproto.PROTO_VERSION),
			MsgType:    proto.Uint32(syncproto.PROTO_MSG_HEART_BETA_RES),
			ContentLen: proto.Uint32(0),
		}
		msgData, err = proto.Marshal(msgRes)
		if err != nil {
			log.Logger.Warn("proto.Marshal heartbeat res msg failed,err:%s", err.Error())
			tmpTry++
			time.Sleep(time.Duration(syncproto.HEART_BEAT_INTERVAL) * time.Second)
			continue
		}
		err = common.WriteMsg(msgData, conn)
		if err != nil {
			tmpTry++
			time.Sleep(time.Duration(syncproto.HEART_BEAT_INTERVAL) * time.Second)
			continue
		}
		// sync file online
		err = syncFileOnline(conn)
		if err != nil {
			log.Logger.Warn("syncFileOnline with %s failed,err:%s", clientAddr, err.Error())
		}
	}

	return nil
}

func StartHeartBeatListener() {
	go func() {
		common.StartListen(fmt.Sprintf(":%d", syncproto.HEART_BEAT_LISTENER_PORT), processHeartBeat)
	}()
}

func syncFiles(conn net.Conn) error {
	// 查找上一次同步时间,如果未同步过则全同步,如果距离上次同步间有部分文件未同步则部分同步

	return nil
}

func syncFileOnline(conn net.Conn) error {
	clientIp := strings.Split(conn.RemoteAddr().String(), ":")[0]
	valid := config.IsInWhiteList(clientIp)
	if !valid {
		return fmt.Errorf("client:%s is not in white list", clientIp)
	}

	syncFlag := false
	online, ok := ClientsAddr[clientIp]
	if !ok || !online {
		ClientsAddr[clientIp] = true
		syncFlag = true
	}
	HeartBeatList[clientIp] = time.Now().Unix()
	if syncFlag {
		go func(conn net.Conn) {
			syncFiles(conn)
		}(conn)
	}
	return nil
}

func sendMsgToClient(ipAddr string, msg *syncproto.FileSyncProto) error {
	log.Logger.Info("send msg to client:%s,msgtype:%d,msgname:%s", ipAddr, msg.GetMsgType(), syncproto.GetMsgName(msg.GetMsgType()))
	msgData, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	conn, err := net.DialTimeout("tcp", ipAddr, common.DIAL_TIMEOUT*time.Second)
	if err != nil {
		return err
	}
	err = common.WriteMsg(msgData, conn)
	if err != nil {
		return err
	}
	// read heartbeat request msg
	msgData, err = common.ReadMsg(conn)
	if err != nil {
		return err
	}
	resMsg := &syncproto.FileSyncProto{}
	err = proto.Unmarshal(msgData, resMsg)
	if err != nil {
		return err
	}
	if resMsg.GetMsgType() != syncproto.PROTO_MSG_COMMON_RESP_OK {
		return fmt.Errorf("client operate failed,resp msg type:%d,msgname:%s", resMsg.GetMsgType(), syncproto.GetMsgName(msg.GetMsgType()))
	}
	return nil
}

func syncCmdPorcess(event fsnotify.Event) error {
	msg := &syncproto.FileSyncProto{
		Version:  proto.Uint32(syncproto.PROTO_VERSION),
		FileName: proto.String(event.Name),
	}

	if event.Op&fsnotify.Create == fsnotify.Create {
		log.Logger.Info("process create:%s", event.Name)
		msg.MsgType = proto.Uint32(syncproto.PROTO_MSG_FILE_CREATE_REQ)
		fi, err := os.Stat(event.Name)
		if err != nil {
			return err
		}
		if fi.IsDir() {
			msg.ContentLen = proto.Uint32(syncproto.PROTO_DIR_LEN)
		} else {
			msg.ContentLen = proto.Uint32(syncproto.PROTO_FILE_LEN)
		}
	} else if event.Op&fsnotify.Write == fsnotify.Write {
		log.Logger.Info("process write:%s", event.Name)
		msg.MsgType = proto.Uint32(syncproto.PROTO_MSG_FILE_WRITE_REQ)
		fileData, err := ioutil.ReadFile(event.Name)
		if err != nil {
			return err
		}
		msg.Content = fileData
		msg.ContentLen = proto.Uint32(uint32(len(fileData)))
	} else if event.Op&fsnotify.Remove == fsnotify.Remove {
		log.Logger.Info("process remove:%s", event.Name)
		msg.MsgType = proto.Uint32(syncproto.PROTO_MSG_FILE_REMOVE_REQ)
		msg.ContentLen = proto.Uint32(0)
	} else if event.Op&fsnotify.Rename == fsnotify.Rename {
		log.Logger.Info("process rename :%s", event.Name)
		msg.MsgType = proto.Uint32(syncproto.PROTO_MSG_FILE_RENAME_REQ)
		msg.ContentLen = proto.Uint32(0)
	} else if event.Op&fsnotify.Chmod == fsnotify.Chmod {
		log.Logger.Info("process chmod file:%s", event.Name)
		msg.MsgType = proto.Uint32(syncproto.PROTO_MSG_FILE_CHMOD_REQ)
		msg.ContentLen = proto.Uint32(0)
	} else {
		log.Logger.Warn("unsupport Op:%d, name:%s", event.Op, event.Name)
		return fmt.Errorf("unsupport Op:%d, name:%s", event.Op, event.Name)
	}

	for clientAddr, t := range HeartBeatList {
		now := time.Now().Unix()
		if now-t > (syncproto.MAX_RETRY_TIME * syncproto.HEART_BEAT_INTERVAL) {
			log.Logger.Info("client:%s lost,not need send msg", clientAddr)
			// record lost file
			continue
		}
		for _, moni := range config.GServerConf.MoniDirs {
			// match moni dir
			if strings.Contains(event.Name, moni.DirName) {
				for _, ipAddr := range moni.WhiteList {
					// match ipaddr
					if clientAddr == strings.Split(ipAddr, ":")[0] {
						go func(ipaddr string, msg *syncproto.FileSyncProto) {
							err := sendMsgToClient(ipaddr, msg)
							if err != nil {
								log.Logger.Error("send msg to client:%s,msgType:%d,msgname:%s failed,err:%s", ipAddr, msg.GetMsgType(), syncproto.GetMsgName(msg.GetMsgType()), err.Error())
							}
						}(ipAddr, msg)
					}
				}
			}
		}
	}

	return nil
}

func startSyncFile() {
	wg := &sync.WaitGroup{}
	for i := 0; i < syncproto.SYNC_FILE_NUM_ONETIME; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for event := range eventChan {
				err := syncCmdPorcess(event)
				if err != nil {
					log.Logger.Error("syncCmdPorcess event,op:%d,file:%s failed,err:%s", event.Op, event.Name, err.Error())
				}
			}
		}()
	}
	wg.Wait()
}

func newFileWatcher(path string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()
	done := make(chan bool)
	go func(baseDir string) {
		for {
			select {
			case event := <-watcher.Events:
				eventChan <- fsnotify.Event{Name: event.Name, Op: event.Op}
			case err := <-watcher.Errors:
				log.Logger.Error("watch [%s] error:%s", baseDir, err.Error())
			}
		}
	}(path)

	err = watcher.Add(path)
	if err != nil {
		return err
	}
	<-done
	return nil
}

func MoniFilesAndSync() error {
	go func() {
		startSyncFile()
	}()

	wg := &sync.WaitGroup{}
	for _, moniDir := range config.GServerConf.MoniDirs {
		wg.Add(1)
		go func(baseDir string) {
			defer wg.Done()
			log.Logger.Info("will monitor dir:%s...", baseDir)
			err := newFileWatcher(baseDir)
			if err != nil {
				log.Logger.Error("newFileWatcher for %s failed,err:%s", baseDir, err.Error())
			}
		}(moniDir.DirName)
	}
	wg.Wait()
	return nil
}
