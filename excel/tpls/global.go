package tpls

import (
	"github.com/iancoleman/strcase"
	"strings"
)

const globalTpl = `package {Package}
import (
	"encoding/json"
)
{Struct}

var {LowerName}Ins = &{Name}{}

func init() {
	registerLoad("{Name}", {LowerName}Ins.load)
}

func (c *{Name}) load(content string) {
	_ = json.Unmarshal([]byte(content), &{LowerName}Ins)
}

func Get{Name}() *{Name} {
	rwlock.Lock()
	defer rwlock.Unlock()
	return {LowerName}Ins
}
`

func GenGlobalFile(packageName, structName, structDefine string) string {
	args := []string{
		"{Package}", packageName,
		"{LowerName}", strcase.ToLowerCamel(structName),
		"{Name}", structName,
		"{Struct}", structDefine,
	}
	r := strings.NewReplacer(args...)
	return r.Replace(globalTpl)
}
