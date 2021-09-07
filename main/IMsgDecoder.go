package main

import (
	"encoding/base64"
	"errors"
	"strings"
)

/*定义消息解码器及其实现*/

type IMsgDecoder interface {
	Decode(line string) (error, IMsg)
}

type tMsgDecoder struct{}

func (m *tMsgDecoder) Decode(line string) (error, IMsg) {
	items := strings.Split(line, " ")
	size := len(items)
	if items[0] == "NAME" && size == 2 {
		name, err := base64.StdEncoding.DecodeString(items[1])
		if err != nil {
			return err, nil
		}

		return nil, &NameMsg{
			Name: string(name),
		}
	}
	if items[0] == "CHAT" && size == 3 {
		name, err := base64.StdEncoding.DecodeString(items[1])
		if err != nil {
			return err, nil
		}

		words, err := base64.StdEncoding.DecodeString(items[2])
		if err != nil {
			return err, nil
		}
		return nil, &ChatMsg{
			Name:  string(name),
			Words: string(words),
		}
	}
	return errors.New("CAN`T CATCH"), nil
}

var MsgDecoder = &tMsgDecoder{}
