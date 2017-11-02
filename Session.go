package network

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

//Session a
type Session struct {
	sid                       int
	disrty                    bool
	server                    *Server
	connection                net.Conn
	incoming                  chan ResIncome
	outgoing                  chan []byte
	reader                    *bufio.Reader
	writer                    *bufio.Writer
	killRoomConnGoroutine     chan bool
	killSocketReaderGoroutine chan bool
	killSocketWriterGoroutine chan bool
	sessionMutex              sync.Mutex
}

//NewSession a
func NewSession(sid int, server *Server, connection net.Conn) *Session {
	writer := bufio.NewWriter(connection)
	reader := bufio.NewReader(connection)

	session := &Session{
		sid:                       sid,
		server:                    server,
		connection:                connection,
		incoming:                  make(chan ResIncome),
		outgoing:                  make(chan []byte),
		reader:                    reader,
		writer:                    writer,
		killRoomConnGoroutine:     make(chan bool),
		killSocketReaderGoroutine: make(chan bool),
		killSocketWriterGoroutine: make(chan bool),
		//		sessionMutex:
	}
	fmt.Println("A new session created. sid=", sid)
	return session
}

func (session *Session) Read() {
	for {
		select {
		case <-session.killSocketReaderGoroutine:
			return
		default:
			buffer := make([]byte, 1024)
			len, err := session.reader.Read(buffer)
			if err != nil {
				if err == io.EOF {
					fmt.Println("Client disconnected. Destroy session, sid=", session.sid)
					session.LeaveAndDelete()
				}

				// else:
				fmt.Println("error reading.")
				fmt.Println(err)
				time.Sleep(100 * time.Millisecond)
				continue
			}
			response := ResIncome{
				sessionID:  session.sid,
				dataLength: len,
				data:       buffer,
			}
			session.incoming <- response
		}
	}
}

func (session *Session) Write() {
	for {
		select {
		case <-session.killSocketWriterGoroutine:
			return
		case data := <-session.outgoing:
			session.writer.Write(data)
			session.writer.Flush()
		}
	}
}

//Listen a
func (session *Session) Listen() {
	go session.Read()
	go session.Write()
}

//LeaveAndDelete a
func (session *Session) LeaveAndDelete() {
	server := *session.server
	sid := session.sid
	server.serverMutex.Lock()
	defer server.serverMutex.Unlock()
	delete(server.sessions, sid)

	// delete

	session.sessionMutex.Lock()
	defer session.sessionMutex.Unlock()

	// release resources

	// resouce: socket reader goroutine & socket writer goroutine
	session.killSocketReaderGoroutine <- true
	session.killSocketWriterGoroutine <- true

	// resource: reader & writer
	session.reader = nil
	session.writer = nil

	// resource: socket conection
	session.connection.Close()
	session.connection = nil

	// resource: RoomConnGoroutine
	session.killRoomConnGoroutine <- true
}
