package monitarr

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
	"sync"

	docker_helper "github.com/hhacker1999/monitarr/internal/adapters/external/docker"
)

type Monitarr struct {
	helper           *docker_helper.Helper
	containers       map[*MonitarrContainer]io.ReadCloser
	logChan          chan MonitarrLog
	lastLogContainer string
	logMutex         *sync.Mutex
	openConnections  map[string]chan struct{}
}

func New(helper *docker_helper.Helper) Monitarr {
	return Monitarr{
		helper:          helper,
		logChan:         make(chan MonitarrLog),
		containers:      make(map[*MonitarrContainer]io.ReadCloser),
		logMutex:        &sync.Mutex{},
		openConnections: make(map[string]chan struct{}),
	}
}

func (m *Monitarr) Run() error {
	containers, err := m.helper.GetContainers()
	if err != nil {
		return err
	}

	for _, v := range containers {
		name := v.Names[0]
		if strings.Contains(name, "psql") || strings.Contains(name, "postgres") {
			continue
		}
		ctr := &MonitarrContainer{
			Id:   v.ID,
			Name: name[1:],
		}
		reader, err := m.helper.GetContianerLogs(ctr.Id, "")
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
				return
			}
			return
		}

		logLength := binary.BigEndian.Uint32(headerBuffer[4:])
		dataBuffer := make([]byte, logLength)
		_, err = reader.Read(dataBuffer)
		if err != nil {
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
