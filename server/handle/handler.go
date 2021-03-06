package handle

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
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

	eventChan    = make(chan fsnotify.Event, syncproto.SYNC_FILE_NUM_ONETIME)
	moniDirNames = make(map[string]chan bool)
	dirRwLock    = sync.RWMutex{}

	ClientsAddr   = make(map[string]bool)
	HeartBeatList = make(map[string]int64)
)

func processHeartBeat(conn net.Conn) error {
	maxTry := syncproto.MAX_RETRY_TIME
	tmpTry := 0
	clientAddr := conn.RemoteAddr().String()
	clientIp := strings.Split(clientAddr, ":")[0]
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
			log.Logger.Warn("not heartbeat msg,resp msg is:%d,%s", msg.GetMsgType(), syncproto.GetMsgName(msg.GetMsgType()))
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
			log.Logger.Warn("WriteMsg failed,err:%s", err.Error())
			tmpTry++
			time.Sleep(time.Duration(syncproto.HEART_BEAT_INTERVAL) * time.Second)
			continue
		}
		HeartBeatList[clientIp] = time.Now().Unix()
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

func fileExist(filename string) bool {
	fileLen, md5, err := common.GetFileInfo(filename)
	if err != nil {
		return false
	}
	msg := &syncproto.FileSyncProto{
		Version:    proto.Uint32(syncproto.PROTO_VERSION),
		MsgType:    proto.Uint32(syncproto.PROTO_MSG_FILE_EXIST_REQ),
		FileName:   proto.String(filename),
		FileMd5:    proto.String(md5),
		ContentLen: proto.Uint32(fileLen),
	}
	err = sendMsgToClients(filename, msg)
	if err != nil {
		return false
	}
	return true
}

func moniDirFunc(dirName string) {
	go func(path string) {
		log.Logger.Info("moni dir:%s", path)
		newFileWatcher(path)
	}(dirName)
}

func syncFiles(conn net.Conn) error {
	clientIp := strings.Split(conn.RemoteAddr().String(), ":")[0]
	moniDir := ""
	// 查找上一次同步时间,如果未同步过则全同步,如果距离上次同步间有部分文件未同步则部分同步
	for _, dir := range config.GServerConf.MoniDirs {
		for _, ip := range dir.WhiteList {
			tmpIp := strings.Split(ip, ":")[0]
			if tmpIp == clientIp {
				moniDir = dir.DirName
				break
			}
		}
	}
	if moniDir == "" {
		log.Logger.Debug("not found dir by ip:%s", clientIp)
		return nil
	}
	log.Logger.Info("will sync dir:%s to client:%s", moniDir, clientIp)
	err := filepath.Walk(moniDir, func(path string, info os.FileInfo, err error) error {
		if path == moniDir {
			return nil
		}
		if info.IsDir() {
			//eventChan <- fsnotify.Event{Name: path, Op: fsnotify.Create}
			moniDirFunc(path)
		} else {
			// check file is exist or not
			if !fileExist(path) {
				eventChan <- fsnotify.Event{Name: path, Op: fsnotify.Write}
			} else {
				log.Logger.Debug("file:[%s] exist in client,not need send", path)
			}
		}

		return nil
	})
	return err
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

	if syncFlag {
		go func(conn net.Conn) {
			syncFiles(conn)
		}(conn)
	}

	return nil
}

func sendMsgToClient(ipAddr string, msg *syncproto.FileSyncProto) error {

	msgData, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	conn, err := net.DialTimeout("tcp", ipAddr, common.DIAL_TIMEOUT*time.Second)
	if err != nil {
		return err
	}
	syncproto.LogMsg(conn, msg)
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
		return fmt.Errorf("client operate failed,resp msg type:%d,msgname:%s", resMsg.GetMsgType(), syncproto.GetMsgName(resMsg.GetMsgType()))
	}
	return nil
}

func sendMsgToClients(fileName string, msg *syncproto.FileSyncProto) error {
	now := time.Now().Unix()
	for clientAddr, t := range HeartBeatList {
		if now-t > (syncproto.MAX_RETRY_TIME * syncproto.HEART_BEAT_INTERVAL) {
			log.Logger.Info("now:%d,preT:%d,client:%s lost,not need send msg", now, t, clientAddr)
			// record lost file
			continue
		}
		for _, moni := range config.GServerConf.MoniDirs {
			// match moni dir
			if strings.Contains(fileName, moni.DirName) {
				for _, ipAddr := range moni.WhiteList {
					// match ipaddr
					if clientAddr == strings.Split(ipAddr, ":")[0] {
						err := sendMsgToClient(ipAddr, msg)
						if err != nil {
							if msg.GetMsgType() != syncproto.PROTO_MSG_FILE_EXIST_REQ {
								log.Logger.Error("send msg to client:%s,msgType:%d,msgname:%s failed,err:%s", ipAddr, msg.GetMsgType(), syncproto.GetMsgName(msg.GetMsgType()), err.Error())
							} else {
								log.Logger.Debug("send msg to client:%s,msgType:%d,msgname:%s failed,err:%s", ipAddr, msg.GetMsgType(), syncproto.GetMsgName(msg.GetMsgType()), err.Error())
							}
							return err
						}
					}
				}
			}
		}
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
			moniDirFunc(event.Name)
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
		dirRwLock.RLock()
		done, ok := moniDirNames[event.Name]
		dirRwLock.RUnlock()
		if ok {
			dirRwLock.Lock()
			done <- true
			delete(moniDirNames, event.Name)
			dirRwLock.Unlock()
			log.Logger.Info("remove/delete moni dir:%s", event.Name)
		}
		msg.MsgType = proto.Uint32(syncproto.PROTO_MSG_FILE_REMOVE_REQ)
		msg.ContentLen = proto.Uint32(0)
	} else if event.Op&fsnotify.Rename == fsnotify.Rename {
		log.Logger.Info("process rename :%s", event.Name)
		dirRwLock.RLock()
		done, ok := moniDirNames[event.Name]
		dirRwLock.RUnlock()
		if ok {
			dirRwLock.Lock()
			done <- true
			delete(moniDirNames, event.Name)
			dirRwLock.Unlock()
			log.Logger.Info("rename/delete moni dir:%s", event.Name)
		}
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

	sendMsgToClients(event.Name, msg)
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
	dirRwLock.RLock()
	_, ok := moniDirNames[path]
	dirRwLock.RUnlock()
	if !ok {
		dirRwLock.Lock()
		moniDirNames[path] = done
		dirRwLock.Unlock()
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
