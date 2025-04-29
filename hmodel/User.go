package hmodel

import (
	"encoding/json"
	"strconv"

	"github.com/DreamvatLab/go/xerr"
	"github.com/DreamvatLab/go/xlog"
)

type User struct {
	ID       string   `json:"sub,omitempty"`
	Username string   `json:"name,omitempty"`
	Email    string   `json:"email,omitempty"`
	Role     int64    `json:"role,omitempty"`
	Level    int32    `json:"level,omitempty"`
	Status   int32    `json:"status,omitempty"`
	Scopes   []string `json:"scope,omitempty"`
}

func (t *User) UnmarshalJSON(d []byte) error {
	type T2 User // create new type with same structure as T but without its method set!
	x := struct {
		T2              // embed
		Role   string   `json:"role,omitempty"`
		Level  string   `json:"level,omitempty"`
		Status string   `json:"status,omitempty"`
		Scopes []string `json:"scope,omitempty"`
	}{T2: T2(*t)} // don't forget this, if you do and 't' already has some fields set you would lose them

	err := json.Unmarshal(d, &x)
	if err != nil {
		return xerr.WithStack(err)
	}

	*t = User(x.T2)
	var status, level int64
	t.Role, err = strconv.ParseInt(x.Role, 10, 64)
	if err != nil {
		xlog.Warn(err)
	}
	status, err = strconv.ParseInt(x.Status, 10, 32)
	if err != nil {
		xlog.Warn(err)
	}
	level, err = strconv.ParseInt(x.Level, 10, 32)
	if err != nil {
		xlog.Warn(err)
	}
	t.Status = int32(status)
	t.Level = int32(level)
	t.Scopes = x.Scopes

	return nil
}
