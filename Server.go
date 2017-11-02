package network

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"sync"

	"github.com/modest-sql/parser"
)

//ResIncome struct of data in
type ResIncome struct {
	sessionID  int
	dataLength int
	data       []byte
}

//ResOutcome e
type ResOutcome struct {
	Type int
	Data string
}

//Server this is a struct
type Server struct {
	sessions     map[int]*Session
	transactions map[int]*Transaction
	lastSid      int
	lastTid      int
	Entrance     chan net.Conn
	Incoming     chan ResIncome
	outgoing     chan ResOutcome
	serverMutex  sync.Mutex
}

//Init initialice a server
func Init() *Server {
	server := &Server{
		sessions:     make(map[int]*Session),
		transactions: make(map[int]*Transaction),
		lastSid:      -1,
		lastTid:      -1,
		Entrance:     make(chan net.Conn),
		Incoming:     make(chan ResIncome),
		outgoing:     make(chan ResOutcome),
		//		serverMutex:
	}
	fmt.Println("A new Server created.")
	return server
}

func (server *Server) createTransaction(sid int, commands []interface{}) {
	server.serverMutex.Lock()
	defer server.serverMutex.Unlock()

	newTransactionID := server.lastTid + 1
	server.lastTid = newTransactionID
	transaction := NewTransaction(newTransactionID, sid, server, commands)

	_, keyExist := server.transactions[newTransactionID]
	if !keyExist {
		server.transactions[newTransactionID] = transaction
	}
}

func (server *Server) printTransactions() {
	for _, transaction := range server.transactions {
		fmt.Println(transaction)
	}
}

//Join c
func (server *Server) Join(connection net.Conn) {
	server.serverMutex.Lock()
	defer server.serverMutex.Unlock()

	newSessionID := server.lastSid + 1
	server.lastSid = newSessionID

	session := NewSession(newSessionID, server, connection)
	session.Listen()
	fmt.Println("session started listening.")

	_, keyExist := server.sessions[newSessionID]
	if !keyExist {
		server.sessions[newSessionID] = session
	}

	go func() {
		for {
			select {
			case <-session.killRoomConnGoroutine:
				return
			case res := <-session.incoming:
				server.Incoming <- res
			}
		}
	}()
}

func (server *Server) handleRequest(response ResIncome) {

	var data ResOutcome
	err := json.Unmarshal(response.data[:response.dataLength], &data)
	if err != nil {
		fmt.Println("Error decoding answer:", err)
	} else {
		fmt.Println(data)
		switch data.Type {
		case 0:
			answer := ResOutcome{0, "_"}
			b, err := json.Marshal(answer)
			if err != nil {
				fmt.Println("Error encoding ping:", err)
			} else {
				server.WriteToSession(response.sessionID, b)
			}
		case 1:
			//to-do metadata
			answer := ResOutcome{1, "{\"DB_Name\": \"Mocca DB\",\"Tables\": [{\"Table_Name\": \"Employee\", \"ColumnNames\": [\"First Name\", \"Second Name\", \"First Last Name\", \"Second Last Name\", \"Age\", \"Married\"],\"ColumnTypes\": [\"char[100]\", \"char[100]\", \"char[100]\", \"char[100]\", \"int\", \"boolean\"]},{\"Table_Name\": \"Department\", \"ColumnNames\": [\"Department Name\", \"Description\", \"Address\"],\"ColumnTypes\": [\"char[100]\", \"char[100]\", \"char[100]\"]}]}"}
			b, err := json.Marshal(answer)
			if err != nil {
				fmt.Println("Error encoding metadata:", err)
			} else {
				server.WriteToSession(response.sessionID, b)
			}
		case 2:
			reader := bytes.NewReader([]byte(data.Data))
			commands, err := parser.Parse(reader)

			if err != nil {
				fmt.Println("err" + string([]byte("[{\"Error\":\""+err.Error()+"\"}]")))
				answer := ResOutcome{4, "[{\"Error\":\"" + err.Error() + "\"}]"}
				b, err := json.Marshal(answer)
				if err != nil {
					fmt.Println("Error encoding error:", err)
				} else {
					server.WriteToSession(response.sessionID, b)
				}
			} else {
				answer := ResOutcome{2, "[{\"Name\":\"Jesus\",\"Age\":\"23\",\"Job\":\"Meme master\"},{\"Name\":\"Soware\",\"Age\":\"22\",\"Job\":\"Asaber\"},{\"Name\":\"FEC\",\"Age\":\"21\",\"Job\":\"Zizizi\"}]"}
				b, err := json.Marshal(answer)
				if err != nil {
					fmt.Println("Error encoding table:", err)
				} else {
					server.WriteToSession(response.sessionID, b)
					server.createTransaction(response.sessionID, commands)
					server.printTransactions()
				}

			}
		default:
			fmt.Println("Fomen Fabian E. Canizales")
		}
	}

}

//Listen c
func (server *Server) Listen() {
	go func() {
		for {
			select {
			case response := <-server.Incoming:
				server.handleRequest(response)
			case conn := <-server.Entrance:
				server.Join(conn)
			}
		}
	}()
}

//WriteToSession a
func (server *Server) WriteToSession(sessionID int, data []byte) {
	for _, session := range server.sessions {
		if session.sid == sessionID {
			session.outgoing <- data
		}
	}
}
