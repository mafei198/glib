package misc

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

func GetAllFiles(path, extension string, cb func(string) error) error {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}
	for _, file := range files {
		if file.IsDir() {
			if err := GetAllFiles(path+"/"+file.Name(), extension, cb); err != nil {
				return err
			}
		} else {
			fullPath := path + "/" + file.Name()
			if filepath.Ext(fullPath) == extension {
				if err := cb(fullPath); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return true
}

func ReadAllFiles(path string, cb func(string, []byte) error) error {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}
	for _, file := range files {
		if file.IsDir() {
			if err := ReadAllFiles(path+"/"+file.Name(), cb); err != nil {
				return err
			}
		} else {
			fullPath := path + "/" + file.Name()
			data, err := ioutil.ReadFile(fullPath)
			if err != nil {
				return err
			}
			if err := cb(fullPath, data); err != nil {
				return err
			}
		}
	}
	return nil
}
