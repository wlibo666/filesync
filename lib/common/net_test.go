package common

import (
	"fmt"
	"net"
	"testing"
	"time"
)

func HandleConn(conn net.Conn) error {
	data, err := ReadMsg(conn)
	if err != nil {
		fmt.Printf("ReadMsg failed,err:%s\n", err.Error())
		return err
	}
	fmt.Printf("read data:%s\n", data)
	return nil
}

func TestStartListen(t *testing.T) {
	go func() {
		StartListen(":6666", HandleConn)
	}()
	time.Sleep(2 * time.Second)

	conn, err := net.Dial("tcp", "192.168.1.100:6666")
	if err != nil {
		t.Fatalf("dial failed,err:%s", err.Error())
	}
	data := "hello world."
	err = WriteMsg([]byte(data), conn)
	if err != nil {
		t.Fatalf("WriteMsg failed,err:%s", err.Error())
	}
	time.Sleep(2 * time.Second)
}
