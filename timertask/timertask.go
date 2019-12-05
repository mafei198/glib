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
package timertask

import (
	"fmt"
	"github.com/go-redis/redis"
	"github.com/mafei198/goslib/gen_server"
	"github.com/mafei198/goslib/logger"
	"github.com/mafei198/goslib/misc"
	"github.com/mafei198/goslib/pool"
	"github.com/rs/xid"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type TimerTask struct {
	pool       *pool.Pool
	taskTicker *time.Ticker
	retry      map[string]int
	serverId   string
	namespace  string
	client     *redis.Client
	handler    Handler
	Option     *Option
}

type Option struct {
	CheckInterval time.Duration
	BatchLimit    int64
}

type Handler func(actorId string, params interface{}) error

func Start(namespace string, client *redis.Client, handler Handler, option ...*Option) (*TimerTask, error) {
	ins := &TimerTask{
		serverId:  "timer_task:" + xid.New().String(),
		namespace: namespace,
		client:    client,
		handler:   handler,
	}
	if len(option) > 0 {
		ins.Option = option[0]
	} else {
		ins.Option = &Option{
			CheckInterval: 1 * time.Second,
			BatchLimit:    100,
		}
	}
	_, err := gen_server.Start(ins.serverId, ins)
	return ins, err
}

func (t *TimerTask) Stop() error {
	return gen_server.Stop(t.serverId, "shutdown")
}

func (t *TimerTask) Exist(key string) bool {
	_, err := t.client.ZScore(t.namespace, key).Result()
	if err == redis.Nil {
		return false
	}
	if err != nil {
		return false
	}
	return true
}

func (t *TimerTask) Add(key string, runAt int64, playerId string, task string) error {
	content := fmt.Sprintf("%s:%s", playerId, task)
	return gen_server.Cast(t.serverId, &AddParams{key, runAt, content})
}

func (t *TimerTask) Update(key string, runAt int64) error {
	return gen_server.Cast(t.serverId, &UpdateParams{key, runAt})
}

func (t *TimerTask) Finish(key string) error {
	return gen_server.Cast(t.serverId, &FinishParams{key})
}

func (t *TimerTask) Del(key string) error {
	return gen_server.Cast(t.serverId, &DelParams{key})
}

func (t *TimerTask) Get(key string) float64 {
	value, err := gen_server.Call(t.serverId, &GetParams{key: key})
	if err != nil {
		return 0
	}
	return value.(float64)
}

var tickerTaskParams = &TickerTaskParams{}

// 初始化一个timer
func (t *TimerTask) Init([]interface{}) (err error) {
	t.pool, err = pool.New(runtime.NumCPU(), func(args interface{}) (interface{}, error) {
		return nil, t.handleTask(args.(string))
	})
	if err != nil {
		return
	}

	// 新建timer
	t.taskTicker = time.NewTicker(t.Option.CheckInterval)
	t.retry = make(map[string]int)
	go func() {
		for range t.taskTicker.C {
			_, err = gen_server.Call(t.serverId, tickerTaskParams)
			if err != nil {
				logger.ERR("timertask tickerTask failed: ", err)
			}
		}
	}()
	return
}

type StopTickerParams struct{}

func (t *TimerTask) HandleCall(req *gen_server.Request) (interface{}, error) {
	if params, ok := req.Msg.(*GetParams); ok {
		return t.get(params.key)
	}
	err := t.handleCallAndCast(req.Msg)
	return nil, err
}

func (t *TimerTask) HandleCast(req *gen_server.Request) {
	_ = t.handleCallAndCast(req.Msg)
}

type FinishParams struct{ key string }
type DelParams struct{ key string }

func (t *TimerTask) handleCallAndCast(msg interface{}) error {
	switch params := msg.(type) {
	case *AddParams:
		return t.handleAdd(params)
	case *UpdateParams:
		return t.handleUpdate(params)
	case *FinishParams:
		t.pool.ProcessAsync(params.key)
		return nil
	case *DelParams:
		return t.del(params.key)
	case *TickerTaskParams:
		t.tickerTask()
	case *StopTickerParams:
		t.taskTicker.Stop()
	}
	return nil
}

func (t *TimerTask) Terminate(reason string) (err error) {
	t.taskTicker.Stop()
	return nil
}

type AddParams struct {
	key     string
	runAt   int64
	content string
}

func (t *TimerTask) handleAdd(params *AddParams) error {
	return t.add(params.key, params.runAt, params.content)
}

type GetParams struct {
	key string
}

func (t *TimerTask) handleGet(params *GetParams) (float64, error) {
	return t.get(params.key)
}

type UpdateParams struct {
	key   string
	runAt int64
}

func (t *TimerTask) handleUpdate(params *UpdateParams) error {
	return t.update(params.key, params.runAt)
}

func mfaKey(key string) string {
	return "timer_task:" + key
}

var MfaExpireDelay int64 = 3600

func (t *TimerTask) add(key string, runAt int64, content string) error {
	mfaExpire := misc.MaxInt64(runAt-time.Now().Unix(), 0) + MfaExpireDelay
	if _, err := t.client.Set(mfaKey(key), content, time.Duration(mfaExpire)*time.Second).Result(); err != nil {
		return err
	}
	member := redis.Z{
		Score:  float64(runAt),
		Member: key,
	}
	if _, err := t.client.ZAdd(t.namespace, member).Result(); err != nil {
		return err
	}
	return nil
}

func (t *TimerTask) update(key string, runAt int64) error {
	score, err := t.client.ZScore(t.namespace, key).Result()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return err
	}
	if score > 0 {
		member := redis.Z{
			Score:  float64(runAt),
			Member: key,
		}
		_, err := t.client.ZAdd(t.namespace, member).Result()
		return err
	}
	return nil
}

func (t *TimerTask) get(key string) (float64, error) {
	score, err := t.client.ZScore(t.namespace, key).Result()
	if err != nil {
		return 0, err
	}
	return score, nil
}

func (t *TimerTask) del(key string) error {
	_, err := t.client.Del(mfaKey(key)).Result()
	if err != nil {
		return err
	}
	_, err = t.client.ZRem(t.namespace, key).Result()
	return err
}

type TickerTaskParams struct{}

// Timer的check帧
func (t *TimerTask) tickerTask() {
	// 取出最近这段check时间内需要执行的task
	opt := redis.ZRangeBy{
		Min:    "0",
		Max:    strconv.Itoa(int(time.Now().Unix())),
		Offset: 0,
		Count:  t.Option.BatchLimit,
	}
	members, err := t.client.ZRangeByScoreWithScores(t.namespace, opt).Result()
	if err != nil {
		logger.ERR("tickerTask failed: ", err)
		return
	}
	// redis中移除要执行的任务，并进行处理
	for _, member := range members {
		key := member.Member.(string)
		t.client.ZRem(t.namespace, key)
		t.pool.ProcessAsync(key)
	}
}

func (t *TimerTask) handleTask(key string) error {
	mfaKey := mfaKey(key)
	content, err := t.client.Get(mfaKey).Result()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return err
	}
	t.client.Del(mfaKey)
	chunks := strings.Split(content, ":")
	playerId := chunks[0]
	task := chunks[1]
	return t.handler(playerId, task)
}
