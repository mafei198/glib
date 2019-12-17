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

package gnet

import (
	"github.com/gorilla/websocket"
	"github.com/mafei198/glib/logger"
	"sync/atomic"
	"time"
)

type WSConn struct {
	mt       int
	conn     *websocket.Conn
	delegate ConnHandler
}

func NewWSConn(conn *websocket.Conn) *WSConn {
	wsConn := new(WSConn)
	wsConn.conn = conn
	return wsConn
}

func (c *WSConn) Start() {
	c.StartReceiveLoop()

	defer func() {
		c.Close("receiveLoop stoped")
	}()
}

func (c *WSConn) StartReceiveLoop() {
	// 在线玩家统计
	defer atomic.AddInt32(&OnlinePlayers, -1)
	atomic.AddInt32(&OnlinePlayers, 1)

	var mt int
	var data []byte
	var err error
	for {
		if !mgr.enableAcceptMsg {
			break
		}
		_ = c.conn.SetReadDeadline(time.Now().Add(ReadTimeout))
		mt, data, err = c.conn.ReadMessage()
		c.mt = mt
		if err != nil {
			break
		}
		if err = c.delegate.OnData(data); err != nil {
			break
		}
	}
	logger.WARN("ws_conn disconnected: ", err)
	c.onClose(err)
}

func (c *WSConn) Close(reason string) error {
	logger.WARN("ws_conn disconnected: ", reason)
	return c.conn.Close()
}

func (c *WSConn) SendData(data []byte) error {
	return c.conn.WriteMessage(c.mt, data)
}

func (c *WSConn) onClose(err error) {
	c.delegate.OnClose(err)
}
