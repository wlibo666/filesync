package handle

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

type S1 struct {
	ServerDirName string `json:"server_dir"`
	LocalDirName  string `json:"local_dir"`
	ServerAddr    string `json:"server_addr"`
}

func myDestFile(filename string) string {
	var S2 []S1 = []S1{
		// windows  linux
		{ServerDirName: "E:\\Tools\\", LocalDirName: "/letv/mysrc"},
		// windows windows
		{ServerDirName: "E:\\WorkCode1\\src\\", LocalDirName: "E:\\WorkCodeBak\\src"},
		// linux windows
		{ServerDirName: "/home/wangchunyan/src/", LocalDirName: "D:\\mysrc"},
		// linux linux
		{ServerDirName: "/home/wangchunyan1/src/", LocalDirName: "/letv/mysrc"},
	}

	for _, dir := range S2 {
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

func TestGetDestFile(t *testing.T) {
	filename := "E:\\Tools\\goenv\\errcheck-12fd1ab9811e54c55207f3e83134ff59829fbf21.zip"
	file := myDestFile(filename)
	t.Logf("file is:%s", file)
	if file != "/letv/mysrc/goenv/errcheck-12fd1ab9811e54c55207f3e83134ff59829fbf21.zip" {
		t.Fatalf("window linux failed")
	}

	filename = "E:\\WorkCode1\\src\\go\\a.go"
	file = myDestFile(filename)
	t.Logf("file is:%s", file)
	if file != "E:\\WorkCodeBak\\src\\go\\a.go" {
		t.Fatalf("windows windows failed")
	}

	filename = "/home/wangchunyan/src/mygo/my.go"
	file = myDestFile(filename)
	t.Logf("file is:%s", file)
	if file != "D:\\mysrc\\mygo\\my.go" {
		t.Fatalf("linux windows failed")
	}

	filename = "/home/wangchunyan1/src/mygo/my.go"
	file = myDestFile(filename)
	t.Logf("file is:%s", file)
	if file != "/letv/mysrc/mygo/my.go" {
		t.Fatalf("linux linux failed")
	}
	fmt.Printf("---\n")
}
