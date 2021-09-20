package src

import (
	"bufio"
	"fmt"
	"io"
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

// 客户端拨号
func DialChatClient(address string) (error, IChatClient) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return err, nil
	}
	return nil, openChatClient(conn, false)
}

// 创建客户端连接
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

func (cli *tChatClient) GetName() string {
	return cli.name
}

func (cli *tChatClient) SetName(name string) {
	cli.name = name
}

// 发送消息
func (cli *tChatClient) Send(msg IMsg) {
	if cli.isClosed() {
		return
	}
	if cli.pendingSend < gMaxPendingSend {
		atomic.AddInt32(&cli.pendingSend, 1)
		cli.sendChan <- msg
	} else {
		cli.dropLogs = append(cli.dropLogs, msg)
	}
}

func (cli *tChatClient) RecvHandler(handler ClientRecvFunc) {
	if cli.isNotClosed() {
		cli.recvHandler = handler
	}
}

func (cli *tChatClient) CloseHandler(handler ClientCloseFunc) {
	if cli.isNotClosed() {
		cli.closeHandler = handler
	}
}

func (cli *tChatClient) Close() {
	if cli.isNotClosed() {
		cli.closeConn()
	}
}

func (cli *tChatClient) open() {
	if !atomic.CompareAndSwapInt32(&cli.openFlag, 0, 1) {
		return
	}
	go cli.beginWrite()
	go cli.beginRead()
}

func (cli *tChatClient) isClosed() bool {
	return cli.closeFlag != 0
}

func (cli *tChatClient) isNotClosed() bool {
	return cli.closeFlag == 0
}

// 触发写，往 conn 中写入数据
func (cli *tChatClient) beginWrite() {
	writer := bufio.NewWriter(cli.conn)
	for {
		select {
		case <-cli.closeChan: // 接收关闭通道的消息，并关系
			_ = cli.conn.Close()
			cli.closeFlag = 2
			cli.postConnClosed()
			return
		case msg := <-cli.sendChan:
			atomic.AddInt32(&cli.pendingSend, -1)
			_, e := writer.WriteString(msg.Encode())
			if e != nil {
				log.Println("tChatClient.beginWriter CLIENT WRITER ERR : ", e)
				cli.closeConn()
				break
			} else {
				cli.sendLogs = append(cli.sendLogs, msg)
			}
		case <-time.After(time.Duration(10) * time.Second):
			cli.postRecvTimeout()
			break
		}
	}
}

func (cli *tChatClient) postRecvTimeout() {
	fmt.Printf("cChatClient.postRecvTicliout, %v, serverFlag=%v\n", cli.name, cli.serverFlag)
	cli.closeConn()
}

// 接收
func (cli *tChatClient) beginRead() {
	reader := bufio.NewReader(cli.conn)
	for {
		data := make([]byte, 10)
		//line, err := reader.ReadString('\n')
		n, err := reader.Read(data)
		line := string(data)
		log.Printf("READ SIZE: %d, data %v", n, line)
		if err != nil {
			if err == io.EOF {
				continue
			}
			log.Println("client read err ", err)
			cli.closeConn() // 发生错误，关闭链接
			break
		}
		err, msg := MsgDecoder.Decode(line)
		if err != nil {
			fn := cli.recvHandler // 获取处理消息方法(回调)
			if fn != nil {
				fn(cli, msg)
			}
			cli.recvLogs = append(cli.recvLogs, msg)
		}
	}
}

func (cli *tChatClient) closeConn() {
	if !atomic.CompareAndSwapInt32(&cli.closeFlag, 0, 1) {
		return
	}
	cli.closeChan <- true
}

func (cli *tChatClient) postConnClosed() {
	fmt.Printf("tChatClient.postConnClosed, %v, serverFlag=%v\n", cli.name, cli.serverFlag)

	handler := cli.closeHandler
	if handler != nil {
		handler(cli)
	}
	cli.closeHandler = nil
	cli.recvHandler = nil
}
