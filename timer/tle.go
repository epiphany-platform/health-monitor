package timer

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/health-monitor/channel"

	"reflect"
	"runtime"
	"time"
)

type (
	// Option Function Definition
	Option func(*TLE)
	// TLE Timer List Element used to manage timers.
	TLE struct {
		Name    string        // Timer name default #default
		Type    int           // Timer type user defined default -1
		SubType int           // Timer subtype user defined default -1
		Key     string        // Timer Key user defined default ""
		C       time.Time     // Timer instance completion nanoseconds
		chnl    reflect.Value // Channel i/o completion
		timer   *time.Timer   // Timer Event
		User    interface{}   // User specfied value
	}
)

const (
	// Completion case statement type
	Completion = "*timer.TLE"
)

// Init initialize TLE with defaults, return address to caller
func (t *TLE) Init() *TLE {
	t.Name = "#default"
	t.Type = -1
	t.SubType = -1
	uuid, _ := uuid.NewRandom()
	t.Key = uuid.String()
	t.chnl = reflect.Value{}
	t.timer = &time.Timer{}
	t.User = nil
	return t
}

// newTLE acquires a new TLE data area with default values, return address to caller
func newTLE() *TLE {
	return new(TLE).Init()
}

// Equal respond true/false whether 2-timer are equal
func (t *TLE) Equal(t2 *TLE) bool {
	return (t.Name == t2.Name &&
		t.Type == t2.Type &&
		t.SubType == t2.SubType &&
		t.Key == t2.Key)
}

// delete stops specified Timer (TLE) and removes Link List element.
func (t *TLE) delete() {
	defer mutex.Unlock()
	mutex.Lock()

	t.chnl.Close()
	release(t)
}

// New Construct and return TLE
func (t *TLE) New() *TLE {
	return &TLE{
		Name:    t.Name,
		Type:    t.Type,
		SubType: t.SubType,
		Key:     t.Key,
		User:    t.User,
	}
}

// Awaitio wait timer completion and send completion via channel to caller.
func (t *TLE) awaitio() {
	select {
	case t.C = <-t.timer.C:
		{
			channel.Send(t.chnl, t.New())
			t.delete()
		}
	}
	runtime.Goexit()
}

// Format returns formatted buffer based upon timer element
func (t *TLE) Format() string {
	return fmt.Sprintf(
		"%s %d %d %s ",
		t.Name,
		t.Type,
		t.SubType,
		t.Key)
}

// IsChannel reports whether t represents a channel value.
func (t *TLE) IsChannel() bool {
	return t.chnl.IsValid() && t.chnl.Kind() == reflect.Chan
}

// Cancel stops removes active timer.
func (t *TLE) Cancel() {
	defer mutex.Unlock()
	mutex.Lock()

	if t.timer != nil && !t.timer.Stop() {
		_, _ = t.chnl.Recv()
	}

	if t.IsChannel() {
		t.chnl.Close()
	}
	release(t)
}
