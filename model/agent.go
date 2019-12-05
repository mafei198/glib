/*
The MIT License (MIT)

Copyright (c) 2018 SavinMax. All rights reserved.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package model

import "go.mongodb.org/mongo-driver/mongo"

type Agent struct {
	TableName string
	Loader    Loader
	Filters   map[int]FilterHandler
}

const (
	FilterAfterLoad = iota
)

type FilterHandler func(ctx, rec interface{})
type Filter struct {
	Type    int
	Handler FilterHandler
}

func NewAgent(table string, loader Loader) *Agent {
	agent := new(Agent)
	agent.TableName = table
	agent.Loader = loader
	agent.Filters = map[int]FilterHandler{}
	RegistLoader(agent)
	return agent
}

func (ins *Agent) Factory(db *mongo.Database) *Model {
	return newModel(ins.TableName, ins.Loader, db)
}

// 加载
func (ins *Agent) Load(ctx Container, rec interface{}) {
	ins.getModel(ctx).Load(ins.Loader.Uuid(rec), rec)
	if handler, ok := ins.Filters[FilterAfterLoad]; ok {
		handler(ctx, rec)
	}
}

// 创建
func (ins *Agent) Create(ctx Container, rec interface{}) {
	ins.getModel(ctx).Create(ins.Loader.Uuid(rec), rec)
}

// 删除
func (ins *Agent) Delete(ctx Container, uuid interface{}) {
	ins.getModel(ctx).Delete(uuid)
}

// 更新
func (ins *Agent) Update(ctx Container, rec interface{}) {
	ins.getModel(ctx).Update(ins.Loader.Uuid(rec), rec)
}

// 查找
func (ins *Agent) Find(ctx Container, uuid interface{}) interface{} {
	return ins.getModel(ctx).Recs[uuid]
}

// 获取Model
func (ins *Agent) getModel(ctx Container) *Model {
	return ctx.GetModel(ins.TableName)
}
