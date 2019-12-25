package tpls

const loadTpl = `
package gd

import (
	"sync"
)

var rwlock = &sync.RWMutex{}

var loaders = map[string]func(string){}

func registerLoad(name string, loader func(content string)) {
	loaders[name] = loader
}

func LoadConfigs(data map[string]string) {
	rwlock.Lock()
	defer rwlock.Unlock()

	for key, content := range data {
		if loader, ok := loaders[key]; ok {
			loader(content)
		}
	}
}`

func GenLoadFile() string {
	return loadTpl
}
