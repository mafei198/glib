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

package broadcast

import (
	"github.com/mafei198/glib/gen_server"
	"github.com/mafei198/glib/logger"
)

type BroadcastMsg struct {
	Channel  string
	SenderId string
	Msg      interface{}
}

type MsgHandler func(subscriber string, msg *BroadcastMsg)

type Broadcast struct {
	Channel     string
	subscribers map[string]MsgHandler
}

func Join(channel, playerId string, handler MsgHandler) error {
	return castChannel(channelKey(channel), &JoinParams{playerId, handler})
}

func Leave(channel, playerId string) error {
	return castChannel(channelKey(channel), &LeaveParams{playerId})
}

func Publish(channel, playerId string, msg interface{}) error {
	message := &BroadcastMsg{
		Channel:  channel,
		SenderId: playerId,
		Msg:      msg,
	}
	return castChannel(channelKey(channel), &PublishParams{message})
}

func castChannel(channel string, msg interface{}) error {
	// 指定的room不存在，创建一个
	if !gen_server.Exists(channel) {
		err := StartChannel(channel)
		if err != nil {
			logger.ERR("start channel failed: ", err)
			return err
		}
	}
	return gen_server.Cast(channel, msg)
}

/*
   GenServer Callbacks
*/
func (b *Broadcast) Init(args []interface{}) (err error) {
	b.Channel = args[0].(string)
	b.subscribers = make(map[string]MsgHandler)
	return nil
}

func (b *Broadcast) HandleCast(req *gen_server.Request) {
	switch params := req.Msg.(type) {
	case *JoinParams:
		b.handleJoin(params)
		break
	case *LeaveParams:
		b.handleLeave(params)
		break
	case *PublishParams:
		b.handlePublish(params)
		break
	}
}

func (b *Broadcast) HandleCall(*gen_server.Request) (interface{}, error) {
	return nil, nil
}

func (b *Broadcast) Terminate(reason string) (err error) {
	b.subscribers = nil
	return nil
}

/*
   Callback Handlers
*/

type JoinParams struct {
	playerId string
	handler  MsgHandler
}

// 玩家加入房间
func (b *Broadcast) handleJoin(params *JoinParams) {
	b.subscribers[params.playerId] = params.handler
	logger.WARN("channel: ", b.Channel, " members: ", len(b.subscribers))
}

type LeaveParams struct{ playerId string }

func (b *Broadcast) handleLeave(params *LeaveParams) {
	if _, ok := b.subscribers[params.playerId]; ok {
		delete(b.subscribers, params.playerId)
	}
}

type PublishParams struct{ msg *BroadcastMsg }

func (b *Broadcast) handlePublish(params *PublishParams) {
	for subscriber, handler := range b.subscribers {
		handler(subscriber, params.msg)
	}
}

func channelKey(channel string) string {
	return "broadcast:" + channel
}
