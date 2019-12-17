package pbmsg

import (
	"fmt"
	"github.com/mafei198/glib/misc"
	"io/ioutil"
	"regexp"
	"strings"
)

func Generate(protoDir, pkg, outfile string) error {
	messages := make([]string, 0)
	err := misc.GetAllFiles(protoDir, ".proto", func(path string) error {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		messages = append(messages, parseMessages(data)...)
		return nil
	})
	if err != nil {
		return err
	}
	return genRegister(pkg, outfile, messages)
}

func genRegister(pkg, outfile string, messages []string) error {
	content := "package " + pkg + "\n\n"
	content += `import (
	"github.com/golang/protobuf/proto"
	"github.com/mafei198/glib/pbmsg"
)`
	content += "\n\nfunc init() {\n"
	for _, message := range messages {
		content += "\t" + genPbMsg(message)
	}
	content += "}"
	return ioutil.WriteFile(outfile, []byte(content), 0644)
}

func genPbMsg(msg string) string {
	format := "pbmsg.Register(func() proto.Message { return new(%s) })\n"
	return fmt.Sprintf(format, msg)
}

func parseMessages(data []byte) []string {
	messages := make([]string, 0)
	exp := regexp.MustCompile("(?m)^message[\\s]+[a-zA-Z0-9_]+[\\s]+{$")
	content := string(data)
	codes := exp.FindAllString(content, -1)
	for _, code := range codes {
		message := parseMessage(code)
		messages = append(messages, message)
	}
	return messages
}

func parseMessage(matched string) string {
	parts := strings.Split(matched, " ")
	for i, part := range parts {
		if i == 0 || part == "" {
			continue
		}
		return part
	}
	panic(matched)
}
