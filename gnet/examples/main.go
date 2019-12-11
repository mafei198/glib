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
package main

import (
	"github.com/mafei198/goslib/gnet"
	"github.com/mafei198/goslib/logger"
	"github.com/mafei198/goslib/misc"
	"github.com/rs/xid"
)

type Agent struct {
	uuid        string
	conn        gnet.Conn
	authed      bool
	accountId   string
	closed      bool
	closeReason error
}

func main() {
	gnet.Start(gnet.ProtocolTCP, "3000", NewAgent)
	logger.INFO("Agent started!")
	misc.WaitForStopSignal(func() {
		logger.INFO("Shutting down net server...")
	})
}

func NewAgent(conn gnet.Conn) gnet.ConnHandler {
	return &Agent{
		uuid: xid.New().String(),
		conn: conn,
	}
}

func (a *Agent) OnData(data []byte) error {
	if !a.authed {
		// TODO auth connection
		a.authed = true
		logger.INFO("auth connection")
		return nil
	}
	// TODO handle message
	logger.INFO("data: ", data)
	return nil
}

func (a *Agent) OnClose(err error) {
	a.closed = true
	a.closeReason = err
}
