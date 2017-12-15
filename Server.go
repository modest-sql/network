package network

import (
	"fmt"
	"net"
	"sync"
)

//Request struct of data in
type Request struct {
	SessionID int64
	Response  Response
}

//ResponseType identifies which server action to execute
type ResponseType int

//Response json struct used for client-server comunication
type Response struct {
	Type ResponseType
	Data string
}

//Depending of the desired server response this would be the possibles actions
const (
	KeepAlive       ResponseType = 200 + iota //200
	NewDatabase                               //201
	LoadDatabase                              //202
	NewTable                                  //203
	FindTable                                 //204
	GetMetadata                               //205
	Query                                     //206
	ShowTransaction                           //207
	Error                                     //208
	Notification                              //209
	SessionExited                             //210
	DropDb                                    //211
)

//Server handles input and output from engine
type Server struct {
	mutex         sync.Mutex
	nextsessionid int64
	sessions      map[int64]*Session

	RequestQueue chan Request
}

//NewServer returns a server instance
func NewServer() *Server {
	server := &Server{
		nextsessionid: 0,
		sessions:      make(map[int64]*Session),
		RequestQueue:  make(chan Request, 100),
	}
	return server
}

//Join adds a session to server
func (server *Server) Join(conn net.Conn) {
	server.mutex.Lock()
	sessionID := server.nextsessionid
	server.nextsessionid++
	session := NewSession(sessionID, server, conn)
	server.sessions[sessionID] = session
	server.mutex.Unlock()
	fmt.Println("New session created ID= ", sessionID)
}

//Send send response to session
func (server *Server) Send(sessionID int64, response Response) {
	server.sessions[sessionID].outgoing <- response
}

//GetSessionsAmount returns amount of current tcp conections
func (server *Server) GetSessionsAmount() int {
	return len(server.sessions)
}
