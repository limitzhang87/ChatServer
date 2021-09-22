package src

import (
	"encoding/base64"
	"strings"
)

/*定义消息解码器及其实现*/

type IMsgDecoder interface {
	Decode(line string) (bool, IMsg)
}

type tMsgDecoder struct{}

func (m *tMsgDecoder) Decode(line string) (bool, IMsg) {
	items := strings.Split(line, " ")
	size := len(items)
	if items[0] == "NAME" && size == 2 {
		name, err := base64.StdEncoding.DecodeString(items[1])
		if err != nil {
			return false, nil
		}

		return true, &NameMsg{
			Name: string(name),
		}
	}
	if items[0] == "CHAT" && size == 3 {
		name, err := base64.StdEncoding.DecodeString(items[1])
		if err != nil {
			return false, nil
		}

		words, err := base64.StdEncoding.DecodeString(items[2])
		if err != nil {
			return false, nil
		}

		return true, &ChatMsg{
			Name:  string(name),
			Words: string(words),
		}
	}

	return false, nil
}

var MsgDecoder = &tMsgDecoder{}
