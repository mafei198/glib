package misc

import (
	"bytes"
	"compress/gzip"
)

func GZip(data []byte) ([]byte, error) {
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
