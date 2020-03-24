package timer

import (
	"reflect"
	"sync"
	"time"
	"github.com/health-monitor/channel"
	"github.com/google/uuid"
)

var (
	mutex sync.Mutex
	pool  = sync.Pool{
		New: func() interface{} {
			return &TLE{
				Name:    "#default",
				Type:    -1,
				SubType: -1,
				Key:     "",
				chnl:    reflect.Value{},
				timer:   &time.Timer{},
			}
		},
	}
)

// Function returns true/false whether acquired from list/heap
func acquire() (tle *TLE, ok bool) {
	tle, ok = pool.Get().(*TLE)
	return
}

// Release object into Pool.
func release(tle *TLE) {
	tle.Init()
	pool.Put(tle)
}

// Type Populate Timer Type of TLE
func Type(TypeID int) Option {
	return func(t *TLE) {
		t.Type = TypeID
	}
}
// User Populate Timer User specified interface{}
func User(UserID interface{}) Option {
	return func(t *TLE) {
		t.User = UserID
	}
}

// SubType populate Timer SubType of TLE
func SubType(SubTypeID int) Option {
	return func(t *TLE) {
		t.SubType = SubTypeID
	}
}

// Name populate Timer Name of TLE
func Name(NameID string) Option {
	return func(t *TLE) {
		t.Name = NameID
	}
}

// Key populate Timer Key of TLE
func Key(KeyID string) Option {
	return func(t *TLE) {
		t.Key = KeyID
	}
}

// Timeout populate Timeout of TLE
func Timeout(TimeoutID int) Option {
	return func(t *TLE) {
		if TimeoutID > 0 {
			t.timer = time.NewTimer(time.Duration(TimeoutID) * time.Second)
		}
	}
}

// Chan populate Chan of TLE
func Chan(ChanID reflect.Value) Option {
	return func(t *TLE) {
		t.chnl = ChanID
	}
}

// Launch a new Timer List Element (TLE) goroutine
func Launch(opts ...Option) (*TLE, bool) {
	defer mutex.Unlock()
	mutex.Lock()

	tle, _ := acquire()

	for _, opt := range opts {
		opt(tle)
	}

	if tle.timer == nil {
		return nil, false
	}

	if tle.Key == "" {
		uuid, _ := uuid.NewRandom()
		tle.Key = uuid.String()
	}
	if !tle.IsChannel() {
		tle.chnl = channel.Make(tle, 1)
	}

	channel.Insert(tle.chnl, reflect.SelectRecv)

	go tle.awaitio()
	return tle, true
}
