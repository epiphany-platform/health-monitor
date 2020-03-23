package channel

import (
	"reflect"
)

// Channel information block
type channels struct {
	usage int
	cases []reflect.SelectCase
}

var ccb channels

// Send Message on channel ch
func Send(ch reflect.Value, message interface{}) bool {
	if ch.Kind() == reflect.Chan {
		ch.Send(reflect.ValueOf(message))
		return true
	}
	return false
}

// Recv Message on channel ch
func Recv(ch reflect.Value) (message interface{}, ok bool) {
	if ch.Kind() == reflect.Chan {
		return ch.Recv()
	}
	return nil, false
}

// Len Function
func Len() int {
	return len(ccb.cases)
}

// Cap Function
func Cap() int {
	return cap(ccb.cases)
}

// Active Function
func Active() bool {
	return Count() > 0
}

// Count Function
func Count() int {
	return ccb.usage
}

// Remove specified channel compress slice
func Remove(idx int) {
	ccb.cases[idx] = ccb.cases[len(ccb.cases)-1]
	ccb.cases[len(ccb.cases)-1] = reflect.SelectCase{}
	ccb.cases = ccb.cases[:len(ccb.cases)-1]
	ccb.usage--
}

// Open function
func Open(Name string, Type interface{}, Depth int) {
	Insert(Make(Type, Depth), reflect.SelectRecv)
}

// Make Function
func Make(rType interface{}, bSize int) reflect.Value {
	return reflect.MakeChan(
		reflect.ChanOf(reflect.BothDir,
			reflect.ValueOf(rType).Type()),
		bSize,
	)
}
func newCase(Chan reflect.Value, Dir reflect.SelectDir) reflect.SelectCase {
	return reflect.SelectCase{
		Dir:  Dir,
		Chan: Chan,
		Send: reflect.Value{},
	}
}

// Insert Function
func Insert(Chan reflect.Value, Dir reflect.SelectDir) {
	ccb.cases = append(ccb.cases, newCase(Chan, Dir))
	ccb.usage++
}

// Awaitio Function
func Awaitio() (reflect.Value, int, reflect.Value, bool) {
	chosen, data, ok := reflect.Select(ccb.cases)
	return ccb.cases[chosen].Chan, chosen, data, ok
}
