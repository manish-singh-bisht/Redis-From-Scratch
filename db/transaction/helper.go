package transactions

import (
	"errors"
	"fmt"
	"sync/atomic"

	RESP "github.com/manish-singh-bisht/Redis-From-Scratch/db/resp"
)

var ErrNestedMulti = errors.New("ERR MULTI calls can not be nested")
var ErrFailedToStart = errors.New("ERR Failed to start transaction")
var ErrQueueWithoutMulti = errors.New("ERR QUEUED without MULTI")
var ErrDiscardWithoutMutli = errors.New("ERR DISCARD without MULTI")
var ErrExecWithoutMulti = errors.New("ERR EXEC without MULTI")

func inconsistency(key string) error {
	return fmt.Errorf("inconsistency detected: global version for key '%s' not found", key)
}

var globalVersion uint64 = 0

// getNextVersion returns a new monotonically increasing version number. what is the limit until this fails??
func getNextVersion() uint64 {
	return atomic.AddUint64(&globalVersion, 1)
}

func (tm *TxManager) queue(clientID string, cmd RESP.RESPMessage, args []RESP.RESPMessage) error {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tx, exists := tm.txs[clientID]
	if !exists || tx.state != txStateStarted {
		return ErrQueueWithoutMulti
	}

	tx.mu.Lock()
	defer tx.mu.Unlock()

	tx.queuedCommands = append(tx.queuedCommands, commandQueued{Cmd: cmd, Args: args})
	return nil
}
