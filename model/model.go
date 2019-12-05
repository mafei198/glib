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

import (
	"errors"
	"github.com/mafei198/goslib/logger"
	"go.mongodb.org/mongo-driver/mongo"
)

type Loader interface {
	UpdateFilter(ctx, uuid interface{}) interface{}
	SelectFilter(ctx interface{}) interface{}
	Factory() interface{}
	Uuid(value interface{}) interface{}
}

type Model struct {
	Loader
	Name       string
	Status     map[interface{}]int
	Recs       map[interface{}]interface{}
	Collection *mongo.Collection
}

const (
	MT_EMPTY = iota
	MT_ORIGIN
	MT_CREATE
	MT_UPDATE
	MT_DELETE
)

func newModel(tableName string, loader Loader, db *mongo.Database) *Model {
	return &Model{
		Loader:     loader,
		Name:       tableName,
		Status:     map[interface{}]int{},
		Recs:       map[interface{}]interface{}{},
		Collection: db.Collection(tableName),
	}
}

// 获取状态
func (ins *Model) GetStatus() map[interface{}]int {
	return ins.Status
}

// 状态清理
func (ins *Model) CleanStatus() {
	ins.Status = map[interface{}]int{}
}

// 加载
func (ins *Model) Load(uuid, rec interface{}) {
	ins.Recs[uuid] = rec
}

// 创建
func (ins *Model) Create(uuid, rec interface{}) {
	ins.updateState(uuid, MT_CREATE)
	ins.Recs[uuid] = rec
}

// 删除
func (ins *Model) Delete(uuid interface{}) {
	if _, ok := ins.Recs[uuid]; ok {
		ins.updateState(uuid, MT_DELETE)
		delete(ins.Recs, uuid)
	}
}

// 更新
func (ins *Model) Update(uuid, rec interface{}) {
	ins.updateState(uuid, MT_UPDATE)
	ins.Recs[uuid] = rec
}

// 查找
func (ins *Model) Find(uuid interface{}) interface{} {
	return ins.Recs[uuid]
}

// 获取状态
func (ins *Model) status(uuid interface{}) int {
	var current int
	if status, ok := ins.Status[uuid]; ok {
		current = status
	} else if _, ok := ins.Recs[uuid]; ok {
		current = MT_ORIGIN
	} else {
		current = MT_EMPTY
	}
	return current
}

// 更新状态
func (ins *Model) updateState(uuid interface{}, state int) {
	current := ins.status(uuid)
	next, err := ins.flowState(current, state)
	// FIXME 测试代码(稳定后取消错误检测)
	if err != nil {
		logger.ERR("updateState failed: ", uuid, current, state)
		panic(err)
	}
	if next == MT_EMPTY {
		delete(ins.Status, uuid)
	} else {
		ins.Status[uuid] = next
	}
}

// 状态流转
var flowStateErr = errors.New("invalid next status")

func (ins *Model) flowState(current, next int) (int, error) {
	switch current {
	case MT_EMPTY:
		switch next {
		case MT_CREATE:
			return MT_CREATE, nil
		case MT_UPDATE:
			return MT_CREATE, nil
		default:
			return 0, flowStateErr
		}
	case MT_ORIGIN:
		switch next {
		case MT_CREATE:
			return MT_UPDATE, nil
		case MT_UPDATE, MT_DELETE:
			return next, nil
		default:
			return 0, flowStateErr
		}
	case MT_CREATE:
		switch next {
		case MT_CREATE, MT_UPDATE:
			return MT_CREATE, nil
		case MT_DELETE:
			return MT_EMPTY, nil
		default:
			return 0, flowStateErr
		}
	case MT_UPDATE:
		switch next {
		case MT_CREATE:
			return MT_UPDATE, nil
		case MT_UPDATE, MT_DELETE:
			return next, nil
		default:
			return 0, flowStateErr
		}
	case MT_DELETE:
		switch next {
		case MT_CREATE:
			return MT_UPDATE, nil
		default:
			return 0, flowStateErr
		}
	}
	return 0, flowStateErr
}
