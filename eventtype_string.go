// generated by stringer --type=EventType; DO NOT EDIT

package bari

import "fmt"

const _EventType_name = "UnknownEventObjectStartEventObjectKeyEventObjectValueEventObjectEndEventArrayStartEventArrayEndEventStringEventNumberEventBooleanEventNullEventEOFEvent"

var _EventType_index = [...]uint8{0, 12, 28, 42, 58, 72, 87, 100, 111, 122, 134, 143, 151}

func (i EventType) String() string {
	if i >= EventType(len(_EventType_index)-1) {
		return fmt.Sprintf("EventType(%d)", i)
	}
	return _EventType_name[_EventType_index[i]:_EventType_index[i+1]]
}
