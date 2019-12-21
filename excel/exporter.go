package excel

import (
	"encoding/json"
	"github.com/mafei198/glib/excel/tpls"
	"github.com/mafei198/glib/misc"
	"github.com/tidwall/pretty"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func ExportGoAndJSON(excelDir, exportSign, jsonExportPath, goExportPath, goPackage string, options ...*Options) error {
	if err := os.MkdirAll(goExportPath, os.ModePerm); err != nil {
		return err
	}
	if err := os.MkdirAll(jsonExportPath, os.ModePerm); err != nil {
		return err
	}

	if !strings.HasSuffix(jsonExportPath, "/") {
		jsonExportPath += "/"
	}
	if !strings.HasSuffix(goExportPath, "/") {
		goExportPath += "/"
	}

	sheetObjects, err := Export(excelDir, exportSign, options...)

	if err != nil {
		panic(err)
	}

	subStructs := "package " + goPackage + "\n"
	for _, s := range sheetObjects {
		if s.IsSub {
			subStructs += s.Define + "\n"
		} else {
			var content string
			if s.IsGlobal {
				content = tpls.GenGlobalFile(goPackage, s.Name, s.Define)
			} else {
				content = tpls.GenConfigFile(goPackage, s.Name, s.Define)
			}
			err := ioutil.WriteFile(goExportPath+s.Name+".go", []byte(content), 0644)
			if err != nil {
				return err
			}
		}

		if !s.IsSub && s.Json != "" {
			err := ioutil.WriteFile(jsonExportPath+s.Name+".json", []byte(s.Json), 0644)
			if err != nil {
				return err
			}
		}
	}
	content := tpls.GenLoadFile()
	err = ioutil.WriteFile(goExportPath+goPackage+".go", []byte(content), 0644)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(goExportPath+goPackage+"_common.go", []byte(subStructs), 0644)
}

func CreateMergeJSON(jsonPath, writePath string, isGzip bool) error {
	datas := map[string]string{}
	err := misc.GetAllFiles(jsonPath, ".json", func(filePath string) error {
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			return err
		}
		name := strings.TrimSuffix(filepath.Base(filePath), ".json")
		datas[name] = string(pretty.Ugly(data))
		return nil
	})
	if err != nil {
		return err
	}
	data, err := json.Marshal(datas)
	if err != nil {
		return err
	}
	if isGzip {
		if data, err = misc.Gzip(data); err != nil {
			return err
		}
	}
	return ioutil.WriteFile(writePath, data, 0644)
}
