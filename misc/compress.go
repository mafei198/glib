package misc

import (
	"bytes"
	"compress/gzip"
	"io"
)

func Gzip(data []byte) ([]byte, error) {
	var in bytes.Buffer
	writer := gzip.NewWriter(&in)
	_, err := writer.Write(data)
	if err != nil {
		return nil, err
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}
	return in.Bytes(), nil
}

func Gunzip(content string) ([]byte, error) {
	reader := bytes.NewBuffer([]byte(content))
	readCloser, err := gzip.NewReader(reader)
	if err != nil {
		return nil, err
	}
	var replyData bytes.Buffer
	_, err = io.Copy(&replyData, readCloser)
	_ = readCloser.Close()
	if err != nil {
		return nil, err
	}
	return replyData.Bytes(), nil
}
