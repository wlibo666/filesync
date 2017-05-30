package common

import (
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/wlibo666/common-lib/log"
)

const (
	MSG_HEADER_LEN = 8
	DIAL_TIMEOUT   = time.Duration(5)
)

type NetHandle func(conn net.Conn) error

func StartListen(addr string, handle NetHandle) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Logger.Error("Listen [%s] failed,err:%s", addr, err.Error())
		os.Exit(1)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Logger.Error("Listener:%s Accept failed,err:%s", addr, err.Error())
			continue
		}
		go func(conn net.Conn) {
			defer conn.Close()
			err := handle(conn)
			if err != nil {
				log.Logger.Warn("handle msg failed,err:%s", err.Error())
			}
		}(conn)
	}
}

func ReadMsg(conn net.Conn) ([]byte, error) {
	dataLen := make([]byte, MSG_HEADER_LEN)
	_, err := conn.Read(dataLen)
	if err != nil {
		return dataLen, err
	}
	msgLen, err := strconv.ParseUint(string(dataLen), 10, 64)
	if err != nil {
		return dataLen, err
	}
	msg := make([]byte, msgLen)
	allReadLen := uint64(0)
	for {
		readLen, err := conn.Read(msg[allReadLen:])
		if err != nil && err != io.EOF {
			return dataLen, err
		}
		if err == io.EOF {
			break
		}
		allReadLen = (allReadLen + uint64(readLen))
		if allReadLen >= msgLen {
			break
		}
	}
	return msg, nil
}

func WriteMsg(data []byte, conn net.Conn) error {
	msgLen := uint64(len(data))
	dataLen := fmt.Sprintf("%08d", msgLen)
	_, err := conn.Write([]byte(dataLen))
	if err != nil {
		return err
	}
	allWriteLen := uint64(0)
	for {
		writeLen, err := conn.Write(data[allWriteLen:])
		if err != nil {
			return err
		}
		allWriteLen = (allWriteLen + uint64(writeLen))
		if allWriteLen >= msgLen {
			break
		}
	}
	return nil
}
