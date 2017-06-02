package handle

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/wlibo666/common-lib/log"
	"github.com/wlibo666/filesync/client/config"
	"github.com/wlibo666/filesync/lib/common"
	syncproto "github.com/wlibo666/filesync/lib/proto"
)

var (
	ERR_ONLY_SUPPORT_HEARTBEAT_MSG = errors.New("Only support heartbeat msg in this port")
)

func StartHeartBeat() {
	for _, syncConf := range config.GClientConf.SyncDirs {
		serverAddr := strings.Split(syncConf.ServerAddr, ":")
		go func(addr string) {
			for {
				serverHeartAddr := fmt.Sprintf("%s:%d", addr, syncproto.HEART_BEAT_LISTENER_PORT)
				conn, err := net.DialTimeout("tcp", serverHeartAddr, common.DIAL_TIMEOUT*time.Second)
				if err != nil {
					log.Logger.Error("Dial for heartbeat to server:%s failed,err:%s", serverHeartAddr, err.Error())
					time.Sleep(time.Duration(syncproto.HEART_BEAT_INTERVAL) * time.Second)
					continue
				}
				for {
					err := HeartBeat(conn)
					if err != nil {
						log.Logger.Warn("HeartBeat with server:%s failed,err:%s", conn.RemoteAddr().String(), err.Error())
						break
					}
					time.Sleep(time.Duration(syncproto.HEART_BEAT_INTERVAL) * time.Second)
				}
			}
		}(serverAddr[0])
	}
}

func HeartBeat(conn net.Conn) error {
	msgReq := &syncproto.FileSyncProto{
		Version:    proto.Uint32(syncproto.PROTO_VERSION),
		MsgType:    proto.Uint32(syncproto.PROTO_MSG_HEART_BETA_REQ),
		ContentLen: proto.Uint32(0),
	}
	msgData, err := proto.Marshal(msgReq)
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
		log.Logger.Warn("ReadMsg from conn:%s failed,err:%s", conn.RemoteAddr().String(), err.Error())
		return err
	}
	msg := &syncproto.FileSyncProto{}
	err = proto.Unmarshal(msgData, msg)
	if err != nil {
		log.Logger.Warn("proto.Unmarshal failed,err:%s", err.Error())
		return err
	}
	syncproto.LogMsg(conn, msg)
	if msg.GetMsgType() != syncproto.PROTO_MSG_HEART_BETA_RES {
		common.WriteMsg([]byte(ERR_ONLY_SUPPORT_HEARTBEAT_MSG.Error()), conn)
		return ERR_ONLY_SUPPORT_HEARTBEAT_MSG
	}

	return nil
}

func getDestFile(filename string) string {
	for _, dir := range config.GClientConf.SyncDirs {
		if strings.Contains(filename, dir.ServerDirName) {
			// server is windows
			if strings.Contains(dir.ServerDirName, "\\") {
				// client is windows
				if strings.Contains(dir.LocalDirName, "\\") {
					return dir.LocalDirName + "\\" + strings.TrimPrefix(filename, dir.ServerDirName)
				} else {
					return filepath.Join(dir.LocalDirName, strings.Replace(strings.TrimLeft(filename, dir.ServerDirName), "\\", "/", -1))
				}
			} else {
				// client is windows
				if strings.Contains(dir.LocalDirName, "\\") {
					return dir.LocalDirName + "\\" + strings.Replace(strings.TrimPrefix(filename, dir.ServerDirName), "/", "\\", -1)
				} else {
					return filepath.Join(dir.LocalDirName, strings.TrimPrefix(filename, dir.ServerDirName))
				}
			}
		}
	}
	return ""
}

func createFile(filename string, dirFlag uint32) error {
	tmpFile := getDestFile(filename)
	if tmpFile == "" {
		return fmt.Errorf("not found dest file by req file:%s", filename)
	}
	if dirFlag == syncproto.PROTO_DIR_LEN {
		log.Logger.Info("now MkdirAll:%s", tmpFile)
		return os.MkdirAll(tmpFile, os.ModePerm)
	} else if dirFlag == syncproto.PROTO_FILE_LEN {
		log.Logger.Info("now OpenFile(create):%s", tmpFile)
		os.MkdirAll(filepath.Dir(tmpFile), os.ModePerm)
		f, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_TRUNC|os.O_RDWR, os.ModePerm)
		if err != nil {
			return err
		}
		defer f.Close()
	}

	return nil
}

func writeFile(filename string, contentLen uint32, data []byte) error {
	tmpFile := getDestFile(filename)
	if tmpFile == "" {
		return fmt.Errorf("not found dest file by req file:%s", filename)
	}
	log.Logger.Info("now OpenFile(write):%s", tmpFile)
	os.MkdirAll(filepath.Dir(tmpFile), os.ModePerm)
	f, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_TRUNC|os.O_RDWR, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()
	n, err := f.Write(data)
	if err != nil {
		return err
	}
	if uint32(n) != contentLen {
		return fmt.Errorf("contentLen:%d,write len:%d,not equal", contentLen, n)
	}
	return nil
}

func removeFile(filename string) error {
	tmpFile := getDestFile(filename)
	if tmpFile == "" {
		return fmt.Errorf("not found dest file by req file:%s", filename)
	}
	log.Logger.Info("now RemoveAll:%s", tmpFile)
	return os.RemoveAll(tmpFile)
}

func renameFile(srcFile, dstFile string) error {
	tmpFile := getDestFile(srcFile)
	if tmpFile == "" {
		return fmt.Errorf("not found dest file by req file:%s", srcFile)
	}

	fi, err := os.Stat(tmpFile)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		log.Logger.Info("now rename:%s is dir,removeall", tmpFile)
		return os.RemoveAll(tmpFile)
	} else {
		log.Logger.Info("now Remove:%s", tmpFile)
		return os.Remove(tmpFile)
	}
	return nil
}

func chmodFile(filename string, mode int) error {
	return nil
}

func fileExist(filename, fileMd5 string, contentLen uint32) error {
	tmpFile := getDestFile(filename)
	os.MkdirAll(filepath.Dir(tmpFile), os.ModePerm)
	fileLen, md5, err := common.GetFileInfo(tmpFile)
	if err != nil {
		return err
	}
	if uint32(fileLen) == contentLen && md5 == fileMd5 {
		return nil
	}
	return fmt.Errorf("server file:%s,md5:%s,len:%d not equl client file:%s,md5:%s,len:%d",
		filename, fileMd5, contentLen, tmpFile, md5, fileLen)
}

func ProcessServer(conn net.Conn) error {
	clientAddr := conn.RemoteAddr().String()
	// read msg from server
	msgData, err := common.ReadMsg(conn)
	if err != nil {
		log.Logger.Warn("ReadMsg from conn:%s failed,err:%s", clientAddr, err.Error())
		return err
	}
	msg := &syncproto.FileSyncProto{}
	err = proto.Unmarshal(msgData, msg)
	if err != nil {
		log.Logger.Warn("proto.Unmarshal failed,err:%s", err.Error())
		return err
	}
	syncproto.LogMsg(conn, msg)

	respMsg := &syncproto.FileSyncProto{
		Version:    proto.Uint32(syncproto.PROTO_VERSION),
		ContentLen: proto.Uint32(0),
	}

	var cmdErr error
	switch msg.GetMsgType() {
	case syncproto.PROTO_MSG_FILE_CREATE_REQ:
		cmdErr = createFile(msg.GetFileName(), msg.GetContentLen())
	case syncproto.PROTO_MSG_FILE_WRITE_REQ:
		cmdErr = writeFile(msg.GetFileName(), msg.GetContentLen(), msg.GetContent())
	case syncproto.PROTO_MSG_FILE_REMOVE_REQ:
		cmdErr = removeFile(msg.GetFileName())
	case syncproto.PROTO_MSG_FILE_RENAME_REQ:
		cmdErr = renameFile(msg.GetFileName(), "dstFile")
	case syncproto.PROTO_MSG_FILE_CHMOD_REQ:
		cmdErr = chmodFile(msg.GetFileName(), 0)
	case syncproto.PROTO_MSG_FILE_EXIST_REQ:
		cmdErr = fileExist(msg.GetFileName(), msg.GetFileMd5(), msg.GetContentLen())
	default:
		return fmt.Errorf("unsupport msgtype:%d", msg.GetMsgType())
	}

	if cmdErr == nil {
		respMsg.MsgType = proto.Uint32(syncproto.PROTO_MSG_COMMON_RESP_OK)
	} else {
		if msg.GetMsgType() != syncproto.PROTO_MSG_FILE_EXIST_REQ {
			log.Logger.Warn("cmdtype:%s failed,err:%s", syncproto.GetMsgName(msg.GetMsgType()), cmdErr.Error())
		} else {
			log.Logger.Debug("cmdtype:%s failed,err:%s", syncproto.GetMsgName(msg.GetMsgType()), cmdErr.Error())
		}
		respMsg.MsgType = proto.Uint32(syncproto.PROTO_MSG_COMMON_RESP_FAIL)
	}
	respData, err := proto.Marshal(respMsg)
	if err != nil {
		log.Logger.Warn("proto.Marshal heartbeat res msg failed,err:%s", err.Error())
		return err
	}
	err = common.WriteMsg(respData, conn)
	if err != nil {
		log.Logger.Warn("WriteMsg failed,err:%s", err.Error())
		return err
	}

	return nil
}
