package network

import (
	"fmt"
)

const (
	active            = 10 // all transactions start active
	failed            = 11
	aborted           = 12
	partiallyCommited = 13
	committed         = 14
	terminated        = 15 // A transaction is said to have terminated if it has either committed or aborted.
)

//Transaction a transaction
type Transaction struct {
	tid      int
	sid      int
	state    int
	server   *Server
	commands []interface{}
}

//NewTransaction a
func NewTransaction(tid int, sid int, server *Server, commands []interface{}) *Transaction {

	transaction := &Transaction{
		tid:      tid,
		sid:      sid,
		server:   server,
		state:    active,
		commands: commands,
	}
	fmt.Println("A new Transaction created. tid=", tid)
	return transaction
}

func (t *Transaction) begin() {
}

func (t *Transaction) getState() int {
	return t.state
}

func (t *Transaction) commit() {

}

func (t *Transaction) rollBack() {

}

func (t *Transaction) wasCommitted() {

}

func (t *Transaction) wasRolledBack() {

}
