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
package player

import (
	"context"
	"github.com/mafei198/goslib/gen_server"
	"github.com/mafei198/goslib/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

type Task struct {
	Content    string
	Version    int64
	NeedExpire bool
}

type Persister struct {
	*GameData
	queue         map[string]*Task
	persistTicker *time.Ticker
}

func startPersister(mgr *GameData) error {
	ins := &Persister{
		GameData:      mgr,
		queue:         map[string]*Task{},
		persistTicker: time.NewTicker(time.Second),
	}
	_, err := gen_server.Start(ins.Uuid, ins)
	return err
}

var remainTask = &RemainTasksParams{}

func ensurePersistered(uuid string) {
	for {
		count, err := gen_server.Call(uuid, remainTask)
		if err == nil && count.(int) == 0 {
			return
		}
		time.Sleep(1 * time.Second)
	}
}

func persistToDB(serverId string, playerId, content string, version int64, needExpire bool) error {
	return gen_server.Cast(serverId, &PersistParams{playerId, &Task{
		Content:    content,
		Version:    version,
		NeedExpire: needExpire,
	}})
}

var ticker = &TickerPersistParams{}

func (p *Persister) Init([]interface{}) (err error) {
	go func() {
		var err error
		for range p.persistTicker.C {
			_, err = gen_server.Call(p.Uuid, ticker)
			if err != nil {
				logger.ERR("persister tickerPersist failed: ", err)
			}
		}
	}()
	return nil
}

type PersistParams struct {
	playerId string
	task     *Task
}

func (p *Persister) HandleCast(req *gen_server.Request) {
	switch params := req.Msg.(type) {
	case *PersistParams:
		p.queue[params.playerId] = params.task
		break
	}
}

type RemainTasksParams struct{}

func (p *Persister) HandleCall(req *gen_server.Request) (interface{}, error) {
	switch req.Msg.(type) {
	case *TickerPersistParams:
		p.tickerPersist()
		break
	case *RemainTasksParams:
		return len(p.queue), nil
	}
	return nil, nil
}

func (p *Persister) Terminate(reason string) (err error) {
	logger.INFO("persister terminate: ", reason)
	return nil
}

type TickerPersistParams struct{}

func (p *Persister) tickerPersist() {
	for playerId, task := range p.queue {
		if err := p.persist(playerId, task); err == nil {
			delete(p.queue, playerId)
			if task.NeedExpire {
				_, err = p.Client.Expire(p.cacheKey(playerId), CacheExpire).Result()
				if err != nil {
					logger.ERR("Persister setexpire failed: ", playerId, err)
				}
			}
		}
	}
}

func (p *Persister) persist(playerId string, task *Task) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	upsert := true
	_, err = p.Collection.UpdateOne(ctx,
		bson.D{
			{"_id", playerId},
			{"UpdatedAt", bson.D{{"$lt", task.Version}}}},
		bson.D{
			{"$set", bson.D{
				{"_id", playerId},
				{"Content", task.Content},
				{"UpdatedAt", task.Version},
			}},
		}, &options.UpdateOptions{Upsert: &upsert})
	if err != nil {
		logger.ERR("Persist data failed: ", playerId, err)
	}
	return
}
