package gossiper

import ("github.com/simonwicky/Peerster/utils"
		_ "encoding/hex"
		_ "fmt"
		_ "os"
		_ "time"
		 "sync"
)

type TLCstorage struct {
	 data map[string] *utils.TLCMessage
	 lock sync.RWMutex
}

func NewTLCstorage() *TLCstorage {
	return &TLCstorage{
		data : make(map[string] *utils.TLCMessage),
	}
}

func (t *TLCstorage) lookupName(name string) bool {
	t.lock.RLock()
	defer t.lock.RUnlock()
	_,ok := t.data[name]
	return ok
}

func (t *TLCstorage) addMessage(msg *utils.TLCMessage) {
	name := msg.TxBlock.Transaction.Name
	t.lock.Lock()
	t.data[name] = msg
	t.lock.Unlock()
}