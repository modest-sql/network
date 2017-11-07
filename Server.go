package network

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"sync"

	"github.com/modest-sql/data"
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

//NewServer server constructor
func NewServer() *Server {
	server := &Server{
		sessions:     make(map[int]*Session),
		transactions: make(map[int]*Transaction),
		lastSid:      -1,
		lastTid:      -1,
		Entrance:     make(chan net.Conn),
		Incoming:     make(chan ResIncome),
		outgoing:     make(chan ResOutcome),
	}
	return server
}

func start() {

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

	var pdata ResOutcome
	err := json.Unmarshal(response.data[:response.dataLength], &pdata)
	if err != nil {
		fmt.Println("Error decoding answer:", err)
	} else {
		fmt.Println(pdata)
		switch pdata.Type {
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
			databaseName := "mock.db"
			type Database struct {
				DbNAme string        `json:"DB_Name"`
				Tables []*data.Table `json:"Tables"`
			}

			database, err := data.LoadDatabase(databaseName)

			if err != nil {
				fmt.Println("Something Happened", err)
			} else {
				tables, err := database.AllTables()
				if err != nil {
					fmt.Println("Something Happened two", err)
				} else {
					datab := Database{databaseName, tables}
					c, err := json.Marshal(datab)
					if err != nil {
						fmt.Println("Error encoding metadata:", err)
					}

					answer := ResOutcome{1, string(c)}
					b, err := json.Marshal(answer)
					if err != nil {
						fmt.Println("Error encoding metadata:", err)
					} else {
						fmt.Println(answer)
						server.WriteToSession(response.sessionID, b)
					}
				}
			}

		case 2:
			reader := bytes.NewReader([]byte(pdata.Data))
			commands, err := parser.Parse(reader)

			if err != nil {
				fmt.Println("err" + "[{\"Error\":\"" + err.Error() + "\"}]")
				answer := ResOutcome{4, "[{\"Error\":\"" + err.Error() + "\"}]"}
				b, err := json.Marshal(answer)
				if err != nil {
					fmt.Println("Error encoding error:", err)
				} else {
					server.WriteToSession(response.sessionID, b)
				}
			} else {
				databaseName := "mock.db"
				db, err := data.LoadDatabase(databaseName)

				if err != nil {
					fmt.Println(err)
				} else {

					resultSet, err := db.ReadTable("MOVIES")
					if err != nil {
						fmt.Println(err)
					}

					str, err := json.Marshal(resultSet.Rows)

					answer := ResOutcome{2, string(str)}
					b, err := json.Marshal(answer)

					if err != nil {
						fmt.Println("Error encoding table:", err)
					} else {
						fmt.Println(answer)
						server.WriteToSession(response.sessionID, b)
						server.createTransaction(response.sessionID, commands)
						server.printTransactions()
					}
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
