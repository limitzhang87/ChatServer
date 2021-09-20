package src

// 定义聊天客户端接口

type IChatClient interface {
	GetName() string
	SetName(name string)

	Send(msg IMsg)
	RecvHandler(handler ClientRecvFunc)
	CloseHandler(handler ClientCloseFunc)

	Close()
}

type ClientRecvFunc func(client IChatClient, msg IMsg)
type ClientCloseFunc func(client IChatClient)
