package main

import (
	"github.com/mafei198/glib/pbmsg"
	"os"
	"path/filepath"
)

func main() {
	protoDir := "./protos"
	pkg := "pt"
	outfile := "./gen/pt/register.go"
	if err := os.MkdirAll(filepath.Dir(outfile), os.ModePerm); err != nil {
		panic(err)
	}
	if err := pbmsg.Generate(protoDir, pkg, outfile); err != nil {
		panic(err)
	}
}
