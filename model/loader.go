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
	"context"
	"github.com/mafei198/goslib/logger"
	"github.com/mafei198/goslib/misc"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

type tables struct {
	MapRec       *mongo.Collection
	MissionEvent *mongo.Collection
}

var Tables tables

var agents = make([]*Agent, 0)

func RegistLoader(agent *Agent) {
	agents = append(agents, agent)
}

var MongoDB *mongo.Database

func LoadAll(ctx Container, db *mongo.Database) error {
	for _, agent := range agents {
		model := agent.Factory(db)
		ctx.AddModel(agent.TableName, model)
		if err := acc(ctx, agent, model); err != nil {
			logger.ERR("Loader load failed: ", agent.TableName, err)
			return err
		}
	}
	misc.PrintMemUsage()
	return nil
}

type Factory func() interface{}
type Collector func(value interface{})

func acc(ctx Container, agent *Agent, model *Model) error {
	loader := agent.Loader
	filter := loader.SelectFilter(ctx)
	if filter == nil {
		return nil
	}

	return BatchLoad(model.Collection, filter, loader.Factory, func(rec interface{}) {
		model.Load(loader.Uuid(rec), rec)
	}, 0)
}

type OnLoad func(interface{})

func BatchLoad(col *mongo.Collection, filter interface{}, factory Factory, onLoad OnLoad, limit int32) error {
	timeout, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	var err error
	var cursor *mongo.Cursor
	if limit > 0 {
		cursor, err = col.Find(timeout, filter, &options.FindOptions{
			BatchSize: &limit,
		})
	} else {
		cursor, err = col.Find(timeout, filter)
	}
	if err != nil {
		return err
	}
	defer cursor.Close(timeout)
	for cursor.Next(timeout) {
		rec := factory()
		if err := cursor.Decode(rec); err != nil {
			return err
		}
		onLoad(rec)
	}
	return cursor.Err()
}
