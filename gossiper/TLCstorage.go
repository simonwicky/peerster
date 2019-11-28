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
	msg,ok := t.data[name]
	return ok && msg.Confirmed != -1
}

func (t *TLCstorage) addMessage(msg *utils.TLCMessage) {
	name := msg.TxBlock.Transaction.Name
	t.lock.Lock()
	t.data[name] = msg
	t.lock.Unlock()
}

func (t *TLCstorage) getConfirmedMessages() []*utils.TLCMessage{
	var confirmed []*utils.TLCMessage
	t.lock.RLock()
	defer t.lock.RUnlock()
	for _,msg := range t.data {
		if msg.Confirmed != -1{
			confirmed = append(confirmed,msg)
		}
	}
	return confirmed
}