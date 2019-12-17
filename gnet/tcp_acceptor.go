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
	"github.com/mafei198/glib/logger"
	"net"
	"strconv"
)

type TcpAcceptor struct {
	host     string
	port     string
	listener net.Listener
	factory  HandlerFactory
}

func init() {
	RegisterAcceptors(ProtocolTCP, &TcpAcceptor{})
}

func (acceptor *TcpAcceptor) Start(port string, factory HandlerFactory) error {
	acceptor.factory = factory
	acceptor.host = ""
	acceptor.port = port
	address := net.JoinHostPort("", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	acceptor.listener = listener

	go acceptor.startAcceptLoop()

	return nil
}

func (acceptor *TcpAcceptor) PrintInfo() {
	AgentPort = strconv.Itoa(acceptor.listener.Addr().(*net.TCPAddr).Port)
	logger.INFO("TcpAgent lis: ", AgentPort)
}

func (acceptor *TcpAcceptor) startAcceptLoop() {
	logger.INFO("Game TCPConn started!")
	for {
		// 新连接
		conn, err := acceptor.listener.Accept()
		//logger.INFO("TcpAcceptor accepted new conn")
		if err != nil {
			logger.ERR("TcpAcceptor accept failed: ", err)
		}

		if !mgr.enableAcceptConn {
			break
		}

		tcpConn := NewTcpConn(conn)
		tcpConn.delegate = acceptor.factory(tcpConn)
		go tcpConn.Start()
	}

	_ = acceptor.listener.Close()
}
