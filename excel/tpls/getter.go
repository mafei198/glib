package tpls

import (
	"github.com/mafei198/glib/misc"
	"regexp"
	"strings"
)

const getterTpl = `package {Package}

import (
	"fmt"
	"encoding/json"
	"github.com/mafei198/glib/logger"
)

{Struct}

type {ContainerName} struct {
	index map[{KeyType}]int32
	list  []*{StructName}
}

var {InstanceName}Ins = &{ContainerName}{index: map[{KeyType}]int32{}}

func init() {
	registerLoad("{StructName}", {InstanceName}Ins.load)
}

func (c *{ContainerName}) load(content string) {
	err := json.Unmarshal([]byte(content), &c.list)
	if err != nil {
	    logger.ERR("load config {ContainerName} failed: ", err)
	}
	for i := 0; i < len(c.list); i++ {
		c.index[c.list[int32(i)].ID] = int32(i)
	}
}

func (c *{ContainerName}) GetItem(key {KeyType}) *{StructName} {
	rwlock.RLock()
	defer rwlock.RUnlock()
	idx, ok := c.index[key]
	if !ok {
	    fmt.Println("config {ContainerName} lookup failed: ", key)
		return nil
	}
	return c.list[idx]
}
func (c *{ContainerName}) GetList() []*{StructName} {
	rwlock.RLock()
	defer rwlock.RUnlock()
	return c.list
}`

var keyExp = regexp.MustCompile(" ID [a-z0-9]+ ")

func GenConfigFile(packageName, structName, structDefine string) string {
	sub := keyExp.FindString(structDefine)
	if sub == "" {
		panic("config structure invalid: " + structDefine)
	}
	keyType := strings.Split(sub, " ")[2]
	args := []string{
		"{Package}", packageName,
		"{ContainerName}", misc.ToLowerCamel(structName),
		"{InstanceName}", strings.TrimPrefix(structName, "Config"),
		"{StructName}", structName,
		"{KeyType}", keyType,
		"{Struct}", structDefine,
	}
	r := strings.NewReplacer(args...)
	return r.Replace(getterTpl)
}
