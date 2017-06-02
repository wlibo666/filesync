package common

import (
	"testing"
)

func TestGetFileInfo(t *testing.T) {
	fileName := "./fileinfo.go"

	fileLen, md5, err := GetFileInfo(fileName)
	if err != nil {
		t.Fatalf("GetFileInfo failed,err:%s", err.Error())
	}
	t.Logf("filename:%s,len:%d,md5:%s", fileName, fileLen, md5)
}
