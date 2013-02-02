package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"os"
)

func md5_sum(str string) string {
	hash := md5.New()
	io.WriteString(hash, str)
	digest := fmt.Sprintf("%x",hash.Sum(nil))
	return digest
}

func readFileToString(path string) (str string, err error) {
	var (
		file *os.File
		part []byte
		prefix bool
	)

	if file,err = os.Open(path); err != nil {
		return
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	buffer := bytes.NewBuffer(make([]byte,0))
	for {
		if part,prefix,err = reader.ReadLine(); err != nil {
			break
		}
		buffer.Write(part)
		if !prefix {
			str = str + buffer.String() + "\n"
			buffer.Reset()
		}
	}
	if err == io.EOF {
		err = nil
	}
	return
}

func searchStrings(item string,arr []string,permissive bool) int {
	var length = len(arr)
	for i := 0; i < length; i++ {
		if item == arr[i] {
			return i
		}
	}
	return -1
}

func Btoi(b bool) int {
	if b { return 1 }
	return 0
}