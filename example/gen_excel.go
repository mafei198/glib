package main

import "github.com/mafei198/glib/excel"

func main() {
	err := excel.ExportGoAndJSON(
		"./example/excels",
		"server",
		"./example/gen/json_files",
		"./example/gen/gd",
		"gd")
	if err != nil {
		panic(err)
	}
	err = excel.CreateMergeJSON("./example/gen/json_files", "./example/gen/configData.json.gz", true)
	if err != nil {
		panic(err)
	}
}
