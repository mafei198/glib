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
	"net"
	"net/http"
	"strconv"
)

type WSConnHandler func(conn *websocket.Conn)

type WSAcceptor struct {
	host     string
	port     string
	listener net.Listener
	factory  HandlerFactory
}

func init() {
	RegisterAcceptors(ProtocolWS, &WSAcceptor{})
}

func (acceptor *WSAcceptor) Start(port string, factory HandlerFactory) error {
	acceptor.factory = factory

	acceptor.host = ""
	acceptor.port = port

	address := net.JoinHostPort(acceptor.host, acceptor.port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	acceptor.listener = listener

	go acceptor.startAcceptLoop()
	return nil
}

func (acceptor *WSAcceptor) PrintInfo() {
	AgentPort = strconv.Itoa(acceptor.listener.Addr().(*net.TCPAddr).Port)
	logger.INFO("TcpAgent lis: ", AgentPort)
}

func (acceptor *WSAcceptor) startAcceptLoop() {
	http.HandleFunc("/", acceptor.wsHandler)
	if err := http.Serve(acceptor.listener, nil); err != nil {
		logger.ERR("start WSConn failed: ", err)
		panic(err)
	}
}

func (acceptor *WSAcceptor) wsHandler(w http.ResponseWriter, r *http.Request) {
	if !mgr.enableAcceptConn {
		return
	}

	logger.INFO("WSConn accepted new conn")
	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.ERR("upgrade:", err)
		return
	}

	wsConn := NewWSConn(conn)
	wsConn.delegate = acceptor.factory(wsConn)
	go wsConn.Start()
}
