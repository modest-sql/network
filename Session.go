package network

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"io"
	"log"
	"math"
	"net"
	"time"
)

const (
	chunkSize = 256
)

//Session stores info from client
type Session struct {
	ID         int64
	server     *Server
	connection net.Conn
	writer     *bufio.Writer
	reader     *bufio.Reader
	outgoing   chan Response
	quitReader chan bool
	quitWriter chan bool
}

//NewSession constructs the session
func NewSession(ID int64, server *Server, connection net.Conn) *Session {
	writer := bufio.NewWriter(connection)
	reader := bufio.NewReader(connection)
	session := &Session{
		ID:         ID,
		server:     server,
		connection: connection,
		writer:     writer,
		reader:     reader,
		outgoing:   make(chan Response),
		quitReader: make(chan bool),
		quitWriter: make(chan bool),
	}
	session.listen()
	return session
}

func (session *Session) leave() {
	server := session.server
	server.RequestQueue <- Request{
		SessionID: session.ID,
		Response:  Response{Type: SessionExited, Data: "-"},
	}

	session.quitWriter <- true
	session.quitReader <- true
	session.connection.Close()
	server.mutex.Lock()
	delete(session.server.sessions, session.ID)
	server.mutex.Unlock()
}

func (session *Session) listen() {
	go session.Read()
	go session.Write()
}

func (session *Session) Read() {
	for {
		select {
		case <-session.quitReader:
			return
		default:

			//Read the length prefix
			prefix := make([]byte, 4)
			readLength, err := session.reader.Read(prefix)
			if err != nil {
				if err == io.EOF {
					log.Println("Client disconnected ID= ", session.ID)
					session.leave()
					break
				}
				log.Println("Error reading message.", err)
				time.Sleep(100 * time.Millisecond)
				continue
			}
			length := int(binary.BigEndian.Uint32(prefix))

			//Read and join the chunks of data
			chunkAmount := int(math.Ceil(float64(length) / float64(chunkSize)))
			message := make([]byte, length)
			chunk := make([]byte, chunkSize)

			for i := 0; i < chunkAmount-1; i++ {
				readLength, err = io.ReadFull(session.reader, chunk)
				if err != nil {
					if err == io.EOF {
						log.Println("Client disconnected ID= ", session.ID)
						session.leave()
						break
					}
					log.Println("Error reading message.", err)
					time.Sleep(100 * time.Millisecond)
					continue
				}
				message = append(message, chunk[:readLength]...)
			}

			lastChunk := make([]byte, length-((chunkAmount-1)*chunkSize))
			readLength, err = io.ReadFull(session.reader, lastChunk)
			message = append(message, lastChunk[:readLength]...)

			if err != nil {
				if err == io.EOF {
					log.Println("Client disconnected ID= ", session.ID)
					session.leave()
					break
				}
				log.Println("Error reading message.", err)
				time.Sleep(100 * time.Millisecond)
				continue
			}

			//Trim excess allocation
			message = message[length:]

			//Decode the json message into a response
			var response Response
			err = json.Unmarshal(message[:length], &response)
			if err != nil {
				log.Println("Error decoding answer:", err)
				continue
			}

			log.Println("Reading", response)
			//Send the response to the server channel
			session.server.RequestQueue <- Request{
				SessionID: session.ID,
				Response:  response,
			}
		}
	}
}

func (session *Session) Write() {
	for {
		select {
		case <-session.quitWriter:
			return
		case response := <-session.outgoing:
			encoded, err := json.Marshal(response)
			if err != nil {
				log.Println("Error encoding:", err)
				continue
			}
			//Write Prefix
			prefix := make([]byte, 4)
			binary.BigEndian.PutUint32(prefix, uint32(len(encoded)))
			_, err = session.writer.Write(prefix)
			if err != nil {
				if err == io.EOF {
					log.Println("Client disconnected ID= ", session.ID)
					session.leave()
					break
				}
			}

			//Write chunks
			chunks := split(encoded, chunkSize)
			for _, chunk := range chunks {
				_, err = session.writer.Write(chunk)
				if err != nil {
					if err == io.EOF {
						log.Println("Client disconnected ID= ", session.ID)
						session.leave()
						break
					}
				}

			}

			log.Println("Sending", response)
			session.writer.Flush()
		}
	}
}

func split(buf []byte, lim int) [][]byte {
	var chunk []byte
	chunks := make([][]byte, 0, len(buf)/lim+1)
	for len(buf) >= lim {
		chunk, buf = buf[:lim], buf[lim:]
		chunks = append(chunks, chunk)
	}
	if len(buf) > 0 {
		chunks = append(chunks, buf[:len(buf)])
	}
	return chunks
}
