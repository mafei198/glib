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
	"github.com/mafei198/goslib/logger"
	"time"
)

var (
	Requested int64
	Responsed int64
)

var OnlinePlayers int32
var AgentPort string

type Network struct{}

type Conn interface {
	SendData(data []byte) error
	Close(reason string) error
}

type ConnHandler interface {
	OnData(data []byte) error
	OnClose(err error)
}

type HandlerFactory func(conn Conn) ConnHandler

type Mgr struct {
	factory          HandlerFactory
	enableAcceptConn bool
	enableAcceptMsg  bool
}

var mgr *Mgr

var acceptors = map[string]Acceptor{}

type Acceptor interface {
	Start(port string, factory HandlerFactory) error
}

const(
	ProtocolTCP = "tcp"
	ProtocolWS = "ws"

	Packet = 4
	ReadTimeout = 60 * time.Second
)

func RegisterAcceptors(protocol string, acceptor Acceptor) {
	acceptors[protocol] = acceptor
}

func Start(protocol, port string, factory HandlerFactory) *Mgr {
	mgr = &Mgr{
		factory:          factory,
		enableAcceptConn: true,
		enableAcceptMsg:  true,
	}
	acceptor := acceptors[protocol]
	if err := acceptor.Start(port, factory); err != nil {
		panic(err)
	}

	go func() {
		for {
			time.Sleep(5 * time.Second)
			logger.INFO("Network Status CCU: ", OnlinePlayers, " Requested: ", Requested, " Responsed: ", Responsed)
		}
	}()

	return mgr
}

func Stop() {
	if mgr != nil {
		mgr.enableAcceptConn = false
		mgr.enableAcceptMsg = false
	}
}
