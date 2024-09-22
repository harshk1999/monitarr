package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"sync"

	docker_helper "github.com/hhacker1999/monitarr/internal/adapters/external/docker"
)

func main() {
	helper := docker_helper.New()

	monitarr := New(&helper)
	err := monitarr.Run()
	if err != nil {
		return
	}

}

type Monitarr struct {
	helper           *docker_helper.Helper
	containers       map[*MonitarrContainer]io.ReadCloser
	logChan          chan MonitarrLog
	lastLogContainer string
	logMutex         *sync.Mutex
}

func New(helper *docker_helper.Helper) Monitarr {
	return Monitarr{
		helper:     helper,
		logChan:    make(chan MonitarrLog),
		containers: make(map[*MonitarrContainer]io.ReadCloser),
		logMutex:   &sync.Mutex{},
	}
}

type MonitarrContainer struct {
	Name string
	Id   string
}

type MonitarrLog struct {
	data          []byte
	containerName string
}

func (m *Monitarr) Run() error {
	containers, err := m.helper.GetContainers()
	if err != nil {
		return err
	}

	for _, v := range containers {
		ctr := &MonitarrContainer{
			Id:   v.ID,
			Name: v.Names[0],
		}
		reader, err := m.helper.GetContianerLogs(ctr.Id)
		if err != nil {
			return err
		}
		m.containers[ctr] = reader
	}

	go m.listenLogs()

	m.spawnLogReaders()

	for {
	}
}

func (m *Monitarr) spawnLogReaders() {
	for key, val := range m.containers {
		key := key
		val := val
		go m.readAndPushLogs(key.Name, val)
	}
}

func (m *Monitarr) readAndPushLogs(name string, reader io.ReadCloser) {
	defer reader.Close()
	headerBuffer := make([]byte, 8)

	for {
		// We want to read Entire header of 8 bytes in a single go
		// If we do not use readfull then we might get half of the header since packet size is
		// arbitary and decided by the network protocol
		_, err := io.ReadFull(reader, headerBuffer)
		if err != nil {
			if err == io.EOF {
				fmt.Println("All logs are read")
				return
			}
			fmt.Println("Error reading in header buffer", err)
			return
		}

		logLength := binary.BigEndian.Uint32(headerBuffer[4:])
		dataBuffer := make([]byte, logLength)
		_, err = reader.Read(dataBuffer)
		if err != nil {
			fmt.Println("Error reading in data buffer", err)
			return
		}

		m.logChan <- MonitarrLog{
			data:          dataBuffer,
			containerName: name,
		}

	}

}

func (m *Monitarr) listenLogs() {
	for {
		select {
		case log := <-m.logChan:
			m.logMutex.Lock()
			if m.lastLogContainer != log.containerName {
				m.lastLogContainer = log.containerName
				fmt.Printf(m.createSeparator(m.lastLogContainer))
				fmt.Printf("\033[0m\n")
			}
			fmt.Printf(string(log.data))
			m.logMutex.Unlock()
		}
	}
}

func (m *Monitarr) createSeparator(name string) string {
	SEP_LEN := 80
	if len(name) > SEP_LEN {
		return fmt.Sprintf("\033[1;35m%s\n", name)
	}

	mid := SEP_LEN / 2
	nameMid := len(name) / 2

	left := mid - nameMid
	str := "\n\n\033[1;35m"

	for i := 0; i < left; i++ {
		str += "-"
	}

	str += name

	for i := left + nameMid; i < SEP_LEN; i++ {
		str += "-"
	}
	str += "\n"

	return str
}
