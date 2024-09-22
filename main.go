package main

import (
	"net/http"

	docker_helper "github.com/hhacker1999/monitarr/internal/adapters/external/docker"
	monitarr "github.com/hhacker1999/monitarr/internal/core/moniarr"
	websocketinterface "github.com/hhacker1999/monitarr/internal/interfaces/websocket"
)

func main() {
	helper := docker_helper.New()

	mtr := monitarr.New(&helper)

	socketInterface := websocketinterface.New(&mtr)

  http.HandleFunc("/ws", socketInterface.HandleConnection)

  http.ListenAndServe(":6842", nil)
}
