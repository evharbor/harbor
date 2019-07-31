package models

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// TypeJSONTime format json time field by myself
type TypeJSONTime struct {
	time.Time
}

// JSONTimeNow return a JSONTime instance init with now time
func JSONTimeNow() TypeJSONTime {

	return TypeJSONTime{Time: time.Now()}
}

// MarshalJSON on JSONTime format Time field with %Y-%m-%d %H:%M:%S
func (t TypeJSONTime) MarshalJSON() ([]byte, error) {
	formatted := fmt.Sprintf("\"%s\"", t.Format("2006-01-02 15:04:05"))
	return []byte(formatted), nil
}

// Value insert timestamp into mysql need this function.
func (t TypeJSONTime) Value() (driver.Value, error) {
	var zeroTime time.Time
	if t.Time.UnixNano() == zeroTime.UnixNano() {
		return nil, nil
	}
	return t.Time, nil
}

// Scan valueof time.Time
func (t *TypeJSONTime) Scan(v interface{}) error {
	value, ok := v.(time.Time)
	if ok {
		*t = TypeJSONTime{Time: value}
		return nil
	}
	return fmt.Errorf("can not convert %v to timestamp", v)
}
