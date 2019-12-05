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
	"container/list"
	"context"
	"github.com/mafei198/goslib/gen_server"
	"github.com/mafei198/goslib/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

const SERVER = "__model_persister__"
const BatchLimit = 100

type Persistor struct{}

type PersistTask struct {
	Model   *Model
	Queries []mongo.WriteModel
}

func PersistAll(ctx Container) error {
	tasks := make([]*PersistTask, 0)
	for _, model := range ctx.GetModels() {
		queries := make([]mongo.WriteModel, 0)
		for id, status := range model.GetStatus() {
			query, err := genQuery(ctx, model, id, status)
			if err != nil {
				return err
			}
			if query != nil {
				queries = append(queries, query)
			}
		}
		if len(queries) > 0 {
			tasks = append(tasks, &PersistTask{
				Model:   model,
				Queries: queries,
			})
		}
	}
	if len(tasks) > 0 {
		return addTask(tasks)
	}
	return nil
}

func CleanAllStatus(ctx Container) {
	for _, model := range ctx.GetModels() {
		model.CleanStatus()
	}
}

func genQuery(ctx Container, model *Model, uuid interface{}, status int) (mongo.WriteModel, error) {
	switch status {
	case MT_CREATE, MT_UPDATE:
		rec := model.Find(uuid)
		if rec == nil {
			return nil, nil
		}
		doc, err := bson.Marshal(rec)
		if err != nil {
			logger.ERR("model persist genQuery failed: ", err)
			return nil, err
		}
		upsert := true
		return &mongo.UpdateOneModel{
			Filter: model.UpdateFilter(ctx, uuid),
			Update: doc,
			Upsert: &upsert,
		}, nil
	case MT_DELETE:
		return &mongo.DeleteOneModel{
			Filter: model.UpdateFilter(ctx, uuid),
		}, nil
	default:
		logger.ERR("Invalid model status: ", uuid, status)
	}
	return nil, nil
}

func (*Persistor) Start() error {
	_, err := gen_server.Start(SERVER, new(Server))
	return err
}

var remainTask = &RemainTasksParams{}

func (*Persistor) Stop() error {
	for {
		count, err := gen_server.Call(SERVER, remainTask)
		if err == nil && count.(int) == 0 {
			break
		}
		time.Sleep(1 * time.Second)
	}
	return nil
}

type AddTaskParams struct {
	Tasks []*PersistTask
}

func addTask(tasks []*PersistTask) error {
	return gen_server.Cast(SERVER, &AddTaskParams{
		Tasks: tasks,
	})
}

type Server struct {
	persistTicker *time.Ticker
	tables        map[string]*mongo.Collection
	tabledList    map[string]*list.List
}

var ticker = &TickerPersistParams{}

func (s *Server) Init([]interface{}) (err error) {
	s.tables = map[string]*mongo.Collection{}
	s.tabledList = map[string]*list.List{}

	s.persistTicker = time.NewTicker(time.Second)
	go func() {
		for range s.persistTicker.C {
			if _, err := gen_server.Call(SERVER, ticker); err != nil {
				logger.ERR("model tickerPersist failed: ", err)
			}
		}
	}()
	return nil
}

type RemainTasksParams struct{}

func (s *Server) HandleCall(req *gen_server.Request) (interface{}, error) {
	switch req.Msg.(type) {
	case *TickerPersistParams:
		s.tickerPersist()
		return nil, nil
	case *RemainTasksParams:
		return s.RemainTasks(), nil
	}
	return nil, nil
}

func (s *Server) HandleCast(req *gen_server.Request) {
	switch params := req.Msg.(type) {
	case *AddTaskParams:
		for _, task := range params.Tasks {
			name := task.Model.Name
			writeList, ok := s.tabledList[name]
			if !ok {
				writeList = list.New()
				s.tabledList[name] = writeList
			}
			if _, ok := s.tables[name]; !ok {
				s.tables[name] = task.Model.Collection
			}
			for _, query := range task.Queries {
				writeList.PushBack(query)
			}
		}
		if s.RemainTasks() >= BatchLimit {
			s.tickerPersist()
		}
		break
	case *TickerPersistParams:
		s.tickerPersist()
		break
	}
}

func (s *Server) Terminate(string) (err error) {
	return nil
}

type TickerPersistParams struct{}

func (s *Server) tickerPersist() {
	if err := s.batchWriteModel(); err != nil {
		logger.ERR("ModelPersister batchWrite failed: ", err)
	}
}

/*
  FIXME MongoDB duplicate key issue: https://jira.mongodb.org/browse/SERVER-14322
*/
func (s *Server) batchWriteModel() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for table, writeList := range s.tabledList {
		models := make([]mongo.WriteModel, 0)
		count := s.batchGet(writeList, func(item interface{}) {
			models = append(models, item.(mongo.WriteModel))
		})
		if count == 0 {
			continue
		}
		_, err := s.tables[table].BulkWrite(ctx, models, options.BulkWrite().SetOrdered(true))
		if err != nil {
			return err
		}
		s.removeList(writeList, count)
	}
	if s.RemainTasks() == 0 {
		return nil
	}
	return gen_server.Cast(SERVER, ticker)
}

type Accumulator func(item interface{})

func (s *Server) batchGet(list *list.List, acc Accumulator) int {
	count := 0
	for item := list.Front(); item != nil; item = item.Next() {
		acc(item.Value)
		count++
		if count >= BatchLimit {
			break
		}
	}

	return count
}

func (s *Server) removeList(list *list.List, count int) {
	for i := 0; i < count; i++ {
		list.Remove(list.Front())
	}
}

func (s *Server) RemainTasks() int {
	count := 0
	for _, writeList := range s.tabledList {
		count += writeList.Len()
	}
	return count
}
