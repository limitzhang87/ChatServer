package src

import (
	"encoding/base64"
	"fmt"
)

/*
定义消息接口，以及相关消息的实现，为方便任意消息内容解码，消息传输时，采用base64转码
*/

type IMsg interface {
	Encode() string
}

type NameMsg struct {
	Name string
}

func (m *NameMsg) Encode() string {
	return fmt.Sprintf("NAME %s\n", base64.StdEncoding.EncodeToString([]byte(m.Name)))
}

type ChatMsg struct {
	Name  string
	Words string
}

func (m *ChatMsg) Encode() string {
	return fmt.Sprintf("CHAT %s %s\n",
		base64.StdEncoding.EncodeToString([]byte(m.Name)),
		base64.StdEncoding.EncodeToString([]byte(m.Words)),
	)
}
