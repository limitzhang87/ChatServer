package test

import (
	"fmt"
	"github.com/limitzhang87/ChatServer/src"
	"log"
	"strings"
	"testing"
	"time"
)

func Test_ChatServer(t *testing.T) {
	fnAssertTrue := func(b bool, msg string) {
		if !b {
			t.Fatal(msg)
		}
	}
	port := 3333
	server := src.NewChatServer()
	err := server.Open(port)
	if err != nil {
		t.Fatal(err)
	}

	clientCount := 3
	address := fmt.Sprintf("localhost:%v", port)
	for i := 0; i < clientCount; i++ {
		err, client := src.DialChatClient(address)
		if err != nil {
			t.Fatal(err)
		}

		id := fmt.Sprintf("c%02d", i)
		client.RecvHandler(func(client src.IChatClient, msg src.IMsg) {
			t.Logf("%v recv: %v\n", id, msg)
		})

		go func() {
			client.SetName(id)
			client.Send(&src.NameMsg{Name: id})

			n := 0
			for range time.Tick(time.Duration(1) * time.Second) {
				client.Send(&src.ChatMsg{
					Name:  id,
					Words: fmt.Sprintf("msg %02d from %v", n, id),
				})
				n++
				if n >= 3 {
					break
				}
			}
			client.Close()
		}()
	}

	// waiting 5 second
	passedSeconds := 0
	for range time.Tick(time.Second) {
		passedSeconds++
		t.Logf("%v seconds passed", passedSeconds)

		if passedSeconds >= 5 {
			break
		}
	}
	log.Println("SERVER CLOSE")
	server.Close()

	logs := server.GetLogs()
	fnHasLog := func(log string) bool {
		for _, it := range logs {
			if strings.Contains(it, log) {
				return true
			}
		}
		return false
	}

	for i := 0; i < clientCount; i++ {
		msg := fmt.Sprintf("tChatServer.handleIncomingConn, clientCount=%v", i+1)
		fnAssertTrue(fnHasLog(msg), "expecting log: "+msg)

		msg = fmt.Sprintf("tChatServer.handleClientClosed, c%02d", i)
		fnAssertTrue(fnHasLog(msg), "expecting log: "+msg)
	}
}
