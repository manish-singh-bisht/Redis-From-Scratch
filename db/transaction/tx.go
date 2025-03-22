package transactions

import (
	"sync"

	RESP "github.com/manish-singh-bisht/Redis-From-Scratch/db/resp"
)

type txState int

const (
	txStateNone txState = iota
	txStateStarted
	txStateAborted
)

type commandQueued struct {
	Cmd  RESP.RESPMessage
	Args []RESP.RESPMessage
}

type tx struct {
	mu             sync.RWMutex
	state          txState
	clientID       string
	queuedCommands []commandQueued
}

func newTx(clientID string) *tx {
	return &tx{
		state:          txStateNone,
		clientID:       clientID,
		queuedCommands: []commandQueued{},
	}
}

type TxManager struct {
	mu            sync.RWMutex
	clientWatches *clientWatches
	txs           map[string]*tx // clientID->transaction
}

func NewTxManager() *TxManager {
	return &TxManager{
		clientWatches: NewClientWatches(getGlobalKeyVersions()),
		txs:           make(map[string]*tx),
	}
}

func (tm *TxManager) Watch(clientID string, key string) {
	tm.clientWatches.startWatch(clientID, key)
}

func (tm *TxManager) Unwatch(clientID string) {
	tm.clientWatches.unwatch(clientID)
}

func (tm *TxManager) Multi(clientID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tx, exists := tm.txs[clientID]
	if exists && tx.state == txStateStarted {
		return ErrNestedMulti
	}

	tx = newTx(clientID)
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.state != txStateNone {
		return ErrFailedToStart
	}

	tx.state = txStateStarted

	tm.txs[clientID] = tx
	return nil
}

func (tm *TxManager) Discard(clientID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tx, exists := tm.txs[clientID]
	if !exists || tx.state != txStateStarted {
		return ErrDiscardWithoutMutli
	}

	tx.mu.Lock()
	defer tx.mu.Unlock()

	delete(tm.txs, clientID)
	return nil
}

func (tm *TxManager) Exec(clientID string) ([]commandQueued, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tx, exists := tm.txs[clientID]
	if !exists || tx.state != txStateStarted {
		return []commandQueued{}, ErrExecWithoutMulti
	}

	commands := tx.queuedCommands

	valid, err := tm.clientWatches.checkWatches(clientID)
	if err != nil || !valid {
		tm.clientWatches.unwatch(clientID) // reset watches even if transaction failed
		delete(tm.txs, clientID)           // remove the transaction and its queued commands
		return []commandQueued{}, err
	}

	// reset watches after EXEC
	tm.clientWatches.unwatch(clientID)
	delete(tm.txs, clientID)
	return commands, nil
}

func (tm *TxManager) GetGlobalKeyVersions(key string) (uint64, bool) {
	return tm.clientWatches.globalKeyVersions.getGlobalVersion(key)
}

func (tm *TxManager) UpdateGlobalKeyVersionsMap(key string) {
	tm.clientWatches.globalKeyVersions.upsertGlobalVersion(key)
}

func (tm *TxManager) Queue(clientID string, cmd RESP.RESPMessage, args []RESP.RESPMessage) error {
	return tm.queue(clientID, cmd, args)
}
