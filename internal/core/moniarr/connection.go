package monitarr

import (
	"context"
	"encoding/binary"
	"io"
)

type LogConnectionInput struct {
	Id   string
	Tail string
}

func (m *Monitarr) SubToConnectionLog(
	id string,
	in LogConnectionInput,
) (chan string, error) {

	outPutChan := make(chan string, 0)
	exitChan := make(chan struct{}, 0)
	exitCtx, exitFunc := context.WithCancel(context.Background())
	reader, err := m.helper.GetContianerLogs(in.Id, in.Tail)
	if err != nil {
		exitFunc()
		return outPutChan, err
	}
	m.logMutex.Lock()
	m.openConnections[id] = exitChan
	m.logMutex.Unlock()
	go m.listenExitConnection(exitFunc, exitChan)
	go m.readLogs(exitCtx, outPutChan, reader)

	return outPutChan, nil
}

func (m *Monitarr) UnSubRequest(id string) {
	m.logMutex.Lock()
	ch, ok := m.openConnections[id]
	if ok {
		ch <- struct{}{}
		close(ch)
		delete(m.openConnections, id)
	}
	m.logMutex.Unlock()
}

func (m *Monitarr) listenExitConnection(exitFunc context.CancelFunc, exitChan chan struct{}) {
	<-exitChan
	exitFunc()
}

func (m *Monitarr) readLogs(ctx context.Context, output chan string, reader io.ReadCloser) {
	defer reader.Close()
	headerBuffer := make([]byte, 8)

	for {
		select {
		case <-ctx.Done():
			close(output)
			break
		default:
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

			output <- string(dataBuffer)
			// m.logChan <- MonitarrLog{
			// 	data:          dataBuffer,
			// 	containerName: name,
			// }
		}
	}
}
