// Copyright 2016 carsonsx. All rights reserved.

package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const VERSION = "0.1"

// 发送文件，此函数会以Multipart form的格式发送文件到指定url
func SendFile(filePath, url string) (code int, body []byte, err error) {
	filename := filepath.Base(filePath)
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)
	_, err = bodyWriter.CreateFormFile("file", filename)
	if err != nil {
		log.Printf("create form failed: %v, filePath: %s, url: %v\n", err, filePath, url)
		return -1, nil, err
	}
	fh, err := os.Open(filePath)
	if err != nil {
		log.Printf("open file failed: %v, filePath: %s, url: %v\n", err, filePath, url)
		return -1, nil, err
	}

	fi, err := fh.Stat()
	if err != nil {
		log.Printf("get file stat failed: %v, filePath: %s, url: %v\n", err, filePath, url)
		return -1, nil, err
	}

	boundary := bodyWriter.Boundary()
	closeBuf := bytes.NewBufferString(fmt.Sprintf("\r\n--%s--\r\n", boundary))
	requestReader := io.MultiReader(bodyBuf, fh, closeBuf)

	req, err := http.NewRequest("POST", url, requestReader)
	if err != nil {
		log.Printf("create request failed: %v, filePath: %s, url: %v\n", err, filePath, url)
		return -1, nil, err
	}
	// Set headers for multipart, and Content Length
	req.Header.Add("Content-Type", "multipart/form-data; boundary="+boundary)
	req.ContentLength = fi.Size() + int64(bodyBuf.Len()) + int64(closeBuf.Len())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("do post failed: %v, filePath: %s, url: %v\n", err, filePath, url)
		return -1, nil, err
	}
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("post file to server failed: %v, filePath: %s, url: %v\n", err, filePath, url)
		return resp.StatusCode, nil, err
	}
	log.Printf(fmt.Sprintf("response from server: url=%s, status=%s, body=%s\n", url, resp.Status, string(body)))
	code = resp.StatusCode
	return
}

func main() {

	path := os.Args[1]
	url := os.Args[2]

	log.Printf("Path: %s\n", path)
	log.Printf("Url: %s\n", url)

	var fi os.FileInfo
	fi, err := os.Lstat(path)
	if err != nil {
		log.Println(err)
		return
	}
	var files []string
	if fi.IsDir() {
		err = filepath.Walk(path, func(filename string, fi os.FileInfo, err error) error { //遍历目录
			if fi.IsDir() {
				return nil
			}
			if strings.Index(filename, "/.svn/") == -1 && strings.Index(filename, "/.git/") == -1 && !strings.HasSuffix(filename, ".DS_Store") {
				files = append(files, filename)
			}
			return nil
		})
	} else {
		files = append(files, path)
	}

	for _, filename := range files {
		upload_url := url
		dir := filepath.Dir(strings.TrimPrefix(filename, path))
		if dir != "." && dir != "/" {
			if !strings.HasSuffix(upload_url, "/") {
				upload_url += "/"
			}
			if strings.HasPrefix(dir, "/") {
				dir = dir[1:]
			}
			upload_url += dir
		}
		_, res, err := SendFile(filename, upload_url)
		if err != nil {
			log.Println(err)
		} else if !strings.HasPrefix(string(res) , "[SUCCESS]") {
			log.Println(string(res))
		} else {
			log.Printf("Successfully uploaded '%s' to %s\n", filename, upload_url)
		}
	}
}
