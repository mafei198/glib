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
package pool

import (
	"container/list"
	"github.com/mafei198/goslib/gen_server"
	"github.com/mafei198/goslib/logger"
)

type Pool struct {
	server *gen_server.GenServer
}

type TaskHandler func(msg interface{}) (interface{}, error)

type Task struct {
	Params interface{}
	Client *gen_server.Request
	Reply  bool
}

type Manager struct {
	tasks       *list.List
	idleWorkers *list.List
	workers     []*Worker
}

func New(size int, handler TaskHandler) (pool *Pool, err error) {
	pool = &Pool{}
	manager := &Manager{
		tasks:       list.New(),
		idleWorkers: list.New(),
		workers:     make([]*Worker, size),
	}
	// init manager
	pool.server, err = gen_server.New(manager, size, handler)
	if err != nil {
		return
	}
	// init workers
	for i := 0; i < size; i++ {
		worker, err := NewWorker(pool, i, handler)
		if err != nil {
			return nil, err
		}
		manager.workers[i] = worker
		manager.idleWorkers.PushBack(worker)
	}

	return
}

func (p *Pool) Process(args interface{}) (interface{}, error) {
	return p.server.ManualCall(&TaskParams{args})
}

func (p *Pool) ProcessAsync(args interface{}) {
	err := p.server.Cast(&TaskParams{args})
	if err != nil {
		logger.ERR("pool ProcessAsync failed: ", err)
	}
}

type ReturnWorkerParams struct{ idx int }

func (p *Pool) ReturnWorker(idx int) {
	err := p.server.Cast(&ReturnWorkerParams{idx})
	if err != nil {
		logger.ERR("pool ReturnWorker failed: ", err)
	}
}

func (m *Manager) Init([]interface{}) (err error) {
	return nil
}

func (m *Manager) HandleCall(req *gen_server.Request) (interface{}, error) {
	switch params := req.Msg.(type) {
	case *TaskParams:
		task := &Task{
			Params: params.Msg,
			Client: req,
			Reply:  true,
		}
		// 取出一个闲置worker处理任务
		worker := m.idleWorkers.Front()
		if worker != nil {
			m.idleWorkers.Remove(worker)
			worker.Value.(*Worker).Process(task)
		} else {
			m.tasks.PushBack(task)
		}
	}
	return nil, nil
}

type TaskParams struct{ Msg interface{} }

func (m *Manager) HandleCast(req *gen_server.Request) {
	switch params := req.Msg.(type) {
	case *TaskParams: // worker处理task
		task := &Task{
			Params: params.Msg,
			Client: req,
			Reply:  false,
		}
		worker := m.idleWorkers.Front()
		if worker != nil {
			m.idleWorkers.Remove(worker)
			worker.Value.(*Worker).Process(task)
		} else {
			m.tasks.PushBack(task)
		}
		break
	case *ReturnWorkerParams: // 取出一个任务由worker执行
		task := m.tasks.Front()
		if task != nil {
			m.tasks.Remove(task)
			task := task.Value.(*Task)
			m.workers[params.idx].Process(task)
		} else {
			m.idleWorkers.PushBack(m.workers[params.idx])
		}
		break
	default:
		logger.ERR("unhandle pool cast: ", params)
	}
}

func (m *Manager) Terminate(reason string) (err error) {
	return nil
}
