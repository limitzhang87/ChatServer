package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"sync/atomic"
	"time"
)

// 聊天客户端，实现 IChatClient 接口
type tChatClient struct {
	conn       net.Conn
	name       string
	openFlag   int32
	closeFlag  int32
	serverFlag bool

	closeChan chan bool
	sendChan  chan IMsg

	sendLogs    []IMsg
	dropLogs    []IMsg
	recvLogs    []IMsg
	pendingSend int32

	recvHandler  ClientRecvFunc
	closeHandler ClientCloseFunc
}

var gMaxPendingSend int32 = 100

func DialChatClient(address string) (error, IChatClient) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return err, nil
	}
	return nil, openChatClient(conn, false)
}

func openChatClient(conn net.Conn, serverFlag bool) IChatClient {
	it := &tChatClient{
		conn:       conn,
		name:       "anonymous",
		openFlag:   0,
		closeFlag:  0,
		serverFlag: serverFlag,
		closeChan:  make(chan bool),
		sendChan:   make(chan IMsg, gMaxPendingSend),
		sendLogs:   []IMsg{},
		dropLogs:   []IMsg{},
		recvLogs:   []IMsg{},
	}
	it.open()
	return it
}

func (me *tChatClient) GetName() string {
	return me.name
}

func (me *tChatClient) SetName(name string) {
	me.name = name
}

func (me *tChatClient) Send(msg IMsg) {
	if me.isClosed() {
		return
	}
	if me.pendingSend < gMaxPendingSend {
		atomic.AddInt32(&me.pendingSend, 1)
		me.sendChan <- msg
	} else {
		me.dropLogs = append(me.dropLogs, msg)
	}
}

func (me *tChatClient) RecvHandler(handler ClientRecvFunc) {
	if me.isNotClosed() {
		me.recvHandler = handler
	}
}

func (me *tChatClient) CloseHandler(handler ClientCloseFunc) {
	if me.isNotClosed() {
		me.closeHandler = handler
	}
}

func (me *tChatClient) Close() {
	if me.isNotClosed() {
		me.closeConn()
	}
}

func (me *tChatClient) open() {
	if !atomic.CompareAndSwapInt32(&me.openFlag, 0, 1) {
		return
	}
	go me.beginWrite()
	go me.beginRead()
}

func (me *tChatClient) isClosed() bool {
	return me.closeFlag != 0
}

func (me *tChatClient) isNotClosed() bool {
	return me.closeFlag == 0
}

func (me *tChatClient) beginWrite() {
	writer := bufio.NewWriter(me.conn)
	for {
		select {
		case <-me.closeChan:
			_ = me.conn.Close()
			me.closeFlag = 2
			me.postConnClosed()
			return
		case msg := <-me.sendChan:
			atomic.AddInt32(&me.pendingSend, -1)
			_, e := writer.WriteString(msg.Encode())
			if e != nil {
				me.closeConn()
				break
			} else {
				me.sendLogs = append(me.sendLogs, msg)
			}
		case <-time.After(time.Duration(10) * time.Second):
			me.postRecvTimeout()
			break
		}
	}
}

func (me *tChatClient) postRecvTimeout() {
	fmt.Printf("cChatClient.postRecvTimeout, %v, serverFlag=%v\n", me.name, me.serverFlag)
	me.closeConn()
}

func (me *tChatClient) beginRead() {
	reader := bufio.NewReader(me.conn)
	for me.isNotClosed() {
		line, err := reader.ReadString('\n')
		if err != nil {
			//me.closeConn()
			log.Println(err)
			break
		}
		err, msg := MsgDecoder.Decode(line)
		if err != nil {
			fn := me.recvHandler
			if fn != nil {
				fn(me, msg)
			}
			me.recvLogs = append(me.recvLogs, msg)
		}
	}
}

func (me *tChatClient) closeConn() {
	if !atomic.CompareAndSwapInt32(&me.closeFlag, 0, 1) {
		return
	}
	me.closeChan <- true
}

func (me *tChatClient) postConnClosed() {
	fmt.Printf("tChatClient.postConnClosed, %v, serverFlag=%v\n", me.name, me.serverFlag)

	handler := me.closeHandler
	if handler != nil {
		handler(me)
	}
	me.closeHandler = nil
	me.recvHandler = nil
}
