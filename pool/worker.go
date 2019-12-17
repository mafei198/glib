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
	"github.com/mafei198/glib/gen_server"
	"github.com/mafei198/glib/logger"
)

func NewWorker(manager *Pool, idx int, handler TaskHandler) (*Worker, error) {
	worker := &Worker{
		idx:     idx,
		manager: manager,
		handler: handler,
	}
	server, err := gen_server.New(worker)
	worker.server = server
	return worker, err
}

type Worker struct {
	idx     int
	manager *Pool
	handler TaskHandler
	server  *gen_server.GenServer
}

func (w *Worker) Process(args interface{}) {
	if err := w.server.Cast(args); err != nil {
		logger.ERR("pool worker process failed: ", err)
	}
}

func (w *Worker) Init(args []interface{}) (err error) {
	return nil
}

func (w *Worker) HandleCall(*gen_server.Request) (interface{}, error) {
	return nil, nil
}

func (w *Worker) HandleCast(req *gen_server.Request) {
	defer w.manager.ReturnWorker(w.idx)
	switch params := req.Msg.(type) {
	case *Task: // 处理定时任务并返回
		result, err := w.handler(params.Params)
		if params.Reply {
			params.Client.Response(result, err)
		}
	}
}

func (w *Worker) Terminate(reason string) (err error) {
	return nil
}
