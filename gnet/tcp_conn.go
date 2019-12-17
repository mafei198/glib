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
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/mafei198/glib/logger"
	"github.com/mafei198/glib/packet"
	"io"
	"math"
	"net"
	"sync/atomic"
	"time"
)

// Block And Receiving "request data"
const MaxIncomingPacket = math.MaxInt16

type TCPConn struct {
	conn     net.Conn
	delegate ConnHandler
}

func NewTcpConn(conn net.Conn) *TCPConn {
	tcpConn := new(TCPConn)
	tcpConn.conn = conn
	return tcpConn
}

func (c *TCPConn) Start() {
	c.StartReceiveLoop()

	defer func() {
		c.cleanup()
	}()
}

func (c *TCPConn) StartReceiveLoop() {
	header := make([]byte, Packet)

	// 在线玩家统计
	defer atomic.AddInt32(&OnlinePlayers, -1)
	atomic.AddInt32(&OnlinePlayers, 1)

	var err error
	var data []byte
	for {
		if !mgr.enableAcceptMsg {
			break
		}
		data, err = c.receive(header)
		if err != nil {
			break
		}
		if err = c.onData(data); err != nil {
			break
		}
	}

	if err != nil {
		//logger.WARN("tcp_conn disconnected: ", err)
	}

	c.onClose(err)
}

// 发送消息
func (c *TCPConn) SendData(data []byte) error {
	writer := packet.Writer()
	writer.WriteInt32((int32)(len(data)))
	writer.WriteRawBytes(data)
	_, err := c.conn.Write(writer.Data())
	return err
}

func (c *TCPConn) Close(reason string) error {
	logger.WARN("tcp_conn disconnected: ", reason)
	return c.conn.Close()
}

// 获取请求数据
func (c *TCPConn) receive(header []byte) ([]byte, error) {
	// 设置读取数据超时时间
	err := c.conn.SetReadDeadline(time.Now().Add(ReadTimeout))
	if err != nil {
		logger.ERR("Receive data timeout: ", err)
		return nil, err
	}

	// 从消息中取最前面4个字节header
	_, err = io.ReadFull(c.conn, header)
	if err != nil {
		//logger.ERR("Receive data head failed: ", err)
		return nil, err
	}

	// header得到消息总长度
	size := binary.BigEndian.Uint32(header)
	if size > MaxIncomingPacket {
		err := fmt.Sprintln("exceed max incomming packet size: ", size)
		logger.ERR(err)
		return nil, errors.New(err)
	}
	// 构建对应消息长度的字节切片用于接收消息
	data := make([]byte, size)
	_, err = io.ReadFull(c.conn, data)
	if err != nil {
		logger.ERR("Receive data body failed: ", err)
		return nil, err
	}
	return data, nil
}

// 清理
func (c *TCPConn) cleanup() {
	_ = c.conn.Close()
}

// 接受消息
func (c *TCPConn) onData(data []byte) error {
	return c.delegate.OnData(data)
}

// 断开连接
func (c *TCPConn) onClose(err error) {
	c.delegate.OnClose(err)
}
