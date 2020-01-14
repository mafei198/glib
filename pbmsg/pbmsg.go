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
package pbmsg

import (
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/mafei198/glib/packet"
	"reflect"
)

var (
	ErrInvalidMsgType  = errors.New("invalid message type")
	ErrUnregisteredMsg = errors.New("unregistered message")
)

type Factory func() proto.Message

var msgFactories = map[string]Factory{}

func Register(factory Factory) {
	name := GetType(factory())
	if _, ok := msgFactories[name]; ok {
		panic(fmt.Sprintln("duplicate message factory", name))
	}
	msgFactories[name] = factory
}

func Encode(pb interface{}) ([]byte, error) {
	if msg, ok := pb.(proto.Message); ok {
		buffer := packet.Writer()
		data, err := proto.Marshal(msg)
		if err != nil {
			return nil, err
		}
		buffer.WriteString(GetType(msg))
		buffer.WriteRawBytes(data)
		return buffer.Data(), err
	}
	return nil, ErrInvalidMsgType
}

func Decode(data []byte) (interface{}, error) {
	buffer := packet.Reader(data)
	name, err := buffer.ReadString()
	if err != nil {
		return nil, err
	}
	factory, ok := msgFactories[name]
	if !ok {
		return nil, ErrUnregisteredMsg
	}
	msg := factory()
	return msg, proto.Unmarshal(buffer.RemainData(), msg)
}

func DecodeWithOut(data []byte, out proto.Message) error {
	buffer := packet.Reader(data)
	_, err := buffer.ReadString()
	if err != nil {
		return err
	}
	return proto.Unmarshal(buffer.RemainData(), out)
}

func GetType(msg interface{}) string {
	if t := reflect.TypeOf(msg); t.Kind() == reflect.Ptr {
		return t.Elem().Name()
	} else {
		return t.Name()
	}
}
