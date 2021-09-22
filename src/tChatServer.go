package src

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"
)

// 聊天服务器IChatServer

type tChatServer struct {
	openFlag  int32
	closeFlag int32

	clients     []IChatClient
	clientCount int
	clientLock  *sync.RWMutex

	listener net.Listener
	recvLogs []IMsg

	logs []string
}

func NewChatServer() IChatServer {
	it := &tChatServer{
		openFlag:    0,
		closeFlag:   0,
		clients:     []IChatClient{},
		clientCount: 0,
		clientLock:  new(sync.RWMutex),
		listener:    nil,
		recvLogs:    []IMsg{},
		logs:        nil,
	}
	return it
}

// make server listener
func (se *tChatServer) Open(port int) error {
	if !atomic.CompareAndSwapInt32(&se.openFlag, 0, 1) {
		return errors.New("server already opened")
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%v", port))
	if err != nil {
		return err
	}
	se.listener = listener
	log.Println("SERVER START")
	go se.beginListening()
	return nil
}

func (se *tChatServer) logf(f string, args ...interface{}) {
	msg := fmt.Sprintf(f, args)
	se.logs = append(se.logs, msg)
	fmt.Println(msg)
}

func (se *tChatServer) GetLogs() []string {
	return se.logs
}

// check server is closed
func (se *tChatServer) isClosed() bool {
	return se.closeFlag != 0
}

// check server is not closed
func (se *tChatServer) isNotClosed() bool {
	return !se.isClosed()
}

// do listen
func (se *tChatServer) beginListening() {
	for !se.isClosed() {
		conn, err := se.listener.Accept()
		if err != nil {
			se.Close()
			break
		}
		se.handleIncomingConn(conn)
	}
}

// global close
func (se *tChatServer) Close() {
	if !atomic.CompareAndSwapInt32(&se.closeFlag, 0, 1) {
		return
	}
	_ = se.listener.Close() // close server
	se.closeAllClients()    // close all clients
}

// client all client
func (se *tChatServer) closeAllClients() {
	se.clientLock.Lock()
	defer se.clientLock.Unlock()

	for i, it := range se.clients {
		if it != nil {
			it.Close()
			se.clients[i] = nil
		}
	}
	se.clientCount = 0
}

// 接收到链接之后处理请求
func (se *tChatServer) handleIncomingConn(conn net.Conn) {
	// init client
	client := openChatClient(conn, true)
	client.RecvHandler(se.handleClientMsg)     // client收消息的回调
	client.CloseHandler(se.handleClientClosed) // client关闭是的回调

	// lock se.clients
	se.clientLock.Lock()
	defer se.clientLock.Unlock()

	// add client to clientPool
	if len(se.clients) > se.clientCount {
		se.clients[se.clientCount] = client
	} else {
		se.clients = append(se.clients, client)
	}
	se.clientCount++

	se.logf("tChatServer.handleIncomingConn, clientCount=%v", se.clientCount)
}

// client 收到消息之后的回调
func (se *tChatServer) handleClientMsg(client IChatClient, msg IMsg) {
	se.recvLogs = append(se.recvLogs, msg)
	if NameMsg, ok := msg.(*NameMsg); ok {
		client.SetName(NameMsg.Name)
	} else if _, ok := msg.(*ChatMsg); ok {
		se.Broadcast(msg)
	}
}

// client 关闭之后的回调
func (se *tChatServer) handleClientClosed(client IChatClient) {
	se.logf("tChatServer.handleClientClosed, %s", client.GetName())

	se.clientLock.Lock()
	defer se.clientLock.Unlock()

	if se.clientCount <= 0 {
		return
	}

	lastI := se.clientCount - 1
	for i, it := range se.clients {
		if it == client {
			if i == lastI {
				se.clients[i] = nil
			} else {
				se.clients[i], se.clients[lastI] = se.clients[i], nil
			}
			se.clientCount--
			break
		}
	}
	se.logf("tChatServer.handleClientClosed, %s, clientCount=%v", client.GetName(), se.clientCount)
}

// 遍历所有client发送消息
func (se *tChatServer) Broadcast(msg IMsg) {
	se.clientLock.RLock()
	defer se.clientLock.RUnlock()
	for _, it := range se.clients {
		if it != nil {
			it.Send(msg)
		}
	}
}
