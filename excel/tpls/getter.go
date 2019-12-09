package tpls

import (
	"github.com/iancoleman/strcase"
	"strings"
)

const getterTpl = `package {Package}

import (
	"fmt"
	"encoding/json"
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
	_ = json.Unmarshal([]byte(content), &c.list)
	for i := 0; i < len(c.list); i++ {
		c.index[c.list[int32(i)].Id] = int32(i)
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

func GenConfigFile(packageName, structName, structDefine, keyType string) string {
	args := []string{
		"{Package}", packageName,
		"{ContainerName}", strcase.ToLowerCamel(structName),
		"{InstanceName}", strings.TrimPrefix(structName, "Config"),
		"{StructName}", structName,
		"{KeyType}", keyType,
		"{Struct}", structDefine,
	}
	r := strings.NewReplacer(args...)
	return r.Replace(getterTpl)
}
