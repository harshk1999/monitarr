package websocketinterface

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	monitarr "github.com/hhacker1999/monitarr/internal/core/moniarr"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WebSocketInterface struct {
	mtr         *monitarr.Monitarr
	mtx         *sync.Mutex
	connections map[*websocket.Conn]*SocketClient
}

type SocketClient struct {
	conn *websocket.Conn
	// NOTE: Key will be contianerid
	subscribedContainers map[string]*SubContianer
}

type SubContianer struct {
	UniqueId   string
	dataChan   chan string
	cancelFunc context.CancelFunc
}

func New(mtr *monitarr.Monitarr) WebSocketInterface {
	return WebSocketInterface{
		connections: make(map[*websocket.Conn]*SocketClient),
		mtr:         mtr,
		mtx:         &sync.Mutex{},
	}
}

func (i *WebSocketInterface) listenForClientMessages(client *SocketClient) {
	conn := client.conn

	type SubInput struct {
		Id      string `json:"id"`
		Tail    string `json:"tail"`
		IsUnSub bool   `json:"is_un_sub"`
	}

	for {
		// Read message from the client
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break
		}
		var input SubInput
		err = json.Unmarshal(message, &input)
		if err != nil {
			fmt.Println("error unmarshalling", err)
			fmt.Println("Garbage message from client", string(message))
			continue
		}

		if len(input.Id) != 0 {
			foundContainer, ok := client.subscribedContainers[input.Id]
			if ok {
				i.mtr.UnSubRequest(foundContainer.UniqueId)
				foundContainer.cancelFunc()
				delete(client.subscribedContainers, input.Id)
			}
			if input.IsUnSub {
				continue
			}

			uniqueId := i.genRandom(20)

			dataChan, err := i.mtr.SubToConnectionLog(uniqueId, monitarr.LogConnectionInput{
				Id:   input.Id,
				Tail: input.Tail,
			})
			if err != nil {
				fmt.Println("Error when subscribing", err)
				continue
			}

			exitCtx, cclFunc := context.WithCancel(context.Background())

			client.subscribedContainers[input.Id] = &SubContianer{
				UniqueId:   uniqueId,
				dataChan:   dataChan,
				cancelFunc: cclFunc,
			}

			go i.listenForLogAndWriteToConnection(exitCtx, client.conn, input.Id, dataChan)
		}

	}
}

func (i *WebSocketInterface) listenForLogAndWriteToConnection(
	ctx context.Context,
	conn *websocket.Conn,
	containerId string,
	dataChan chan string,
) {

	for {
		select {
		case <-ctx.Done():
			break
		case log := <-dataChan:
			message := map[string]interface{}{
				"container_id": containerId,
				"log":          log,
			}
			response, _ := json.Marshal(message)
			err := conn.WriteMessage(websocket.TextMessage, response)
			if err != nil {
				fmt.Println("Error writing message to client", err)
			}
		}
	}

}

func (i *WebSocketInterface) HandleConnection(w http.ResponseWriter, r *http.Request) {
	fmt.Println("New Connection Request Received")
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Failed to upgrade to WebSocket:", err)
		return
	}
	i.mtx.Lock()
	client := &SocketClient{
		conn:                 conn,
		subscribedContainers: make(map[string]*SubContianer),
	}
	i.connections[conn] = client
	i.mtx.Unlock()
	go i.listenForClientMessages(client)
}

func (i *WebSocketInterface) genRandom(length int) string {
	input := "abcdefghijklmnopqrstuvwxyz124567890"
	var str string
	for i := 0; i < length; i++ {

		randomVal := rand.Float64()

		index := float64(len(input)) * randomVal

		str += string(input[int(index)])
	}

	return str
}
