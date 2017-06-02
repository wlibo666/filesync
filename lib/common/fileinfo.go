package common

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
)

//return fileLen,Md5, error
func GetFileInfo(filename string) (uint32, string, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return uint32(0), "", err
	}
	sum := md5.Sum(data)
	fileMd5 := ""
	for i := 0; i < md5.Size; i++ {
		fileMd5 += fmt.Sprintf("%02x", sum[i])
	}
	return uint32(len(data)), fileMd5, nil
}
