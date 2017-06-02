package proto

import (
	"net"

	"github.com/wlibo666/common-lib/log"
)

var (
	PROTO_VERSION = uint32(1)

	PROTO_MSG_FILE_CREATE_REQ = uint32(1001)
	PROTO_MSG_FILE_WRITE_REQ  = uint32(1002)
	PROTO_MSG_FILE_REMOVE_REQ = uint32(1003)
	PROTO_MSG_FILE_RENAME_REQ = uint32(1004)
	PROTO_MSG_FILE_CHMOD_REQ  = uint32(1005)
	PROTO_MSG_FILE_EXIST_REQ  = uint32(1006)

	PROTO_MSG_COMMON_RESP_OK   = uint32(2000)
	PROTO_MSG_COMMON_RESP_FAIL = uint32(2001)
	PROTO_MSG_HEART_BETA_REQ   = uint32(3000)
	PROTO_MSG_HEART_BETA_RES   = uint32(3001)

	PROTO_DIR_LEN  = uint32(0)
	PROTO_FILE_LEN = uint32(1)
)

const (
	SYNC_FILE_NUM_ONETIME    = 30
	MAX_RETRY_TIME           = 30
	HEART_BEAT_INTERVAL      = 10
	HEART_BEAT_LISTENER_PORT = 6001
)

func GetMsgName(msgType uint32) string {
	switch msgType {
	case PROTO_MSG_FILE_CREATE_REQ:
		return "createReq"
	case PROTO_MSG_FILE_WRITE_REQ:
		return "writeReq"
	case PROTO_MSG_FILE_REMOVE_REQ:
		return "removeReq"
	case PROTO_MSG_FILE_RENAME_REQ:
		return "renameReq"
	case PROTO_MSG_FILE_CHMOD_REQ:
		return "chmodReq"
	case PROTO_MSG_FILE_EXIST_REQ:
		return "existReq"
	case PROTO_MSG_COMMON_RESP_OK:
		return "respOk"
	case PROTO_MSG_COMMON_RESP_FAIL:
		return "respFail"
	case PROTO_MSG_HEART_BETA_REQ:
		return "heartBeatReq"
	case PROTO_MSG_HEART_BETA_RES:
		return "heartBeatResp"
	default:
		return "unknownMsg"
	}
	return "unknownMsg"
}

func LogMsg(conn net.Conn, msg *FileSyncProto) {
	log.Logger.Debug("conn:%s,version:%d,msgType:%d,msgName:%s,filename:%s,filemd5:%s,contentLen:%d", conn.RemoteAddr().String(),
		msg.GetVersion(), msg.GetMsgType(), GetMsgName(msg.GetMsgType()), msg.GetFileName(), msg.GetFileMd5(), msg.GetContentLen())
}
