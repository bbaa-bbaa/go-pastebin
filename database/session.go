// Copyright 2024 bbaa
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package database

import (
	"encoding/json"
	"reflect"
	"time"

	"github.com/google/uuid"
)

type Session struct {
	UUID string
	data map[string]any
}

func NewSession() (session *Session, err error) {
	session_uuid, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	session = &Session{
		UUID: session_uuid.String(),
		data: make(map[string]any),
	}
	return
}

func GetSession(uuid string) (session *Session, err error) {
	row, err := db.Queryx("SELECT name, value FROM sessions WHERE uuid = ?", uuid)
	if err != nil {
		return nil, err
	}
	session = &Session{
		UUID: uuid,
		data: make(map[string]any),
	}
	if row.Next() {
		var name string
		var value json.RawMessage
		err = row.Scan(&name, &value)
		if err != nil {
			return nil, err
		}
		session.data[name] = value
	}
	return session, nil
}

func (s *Session) Set(name string, value any, duration time.Duration) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = db.Exec("INSERT OR REPLACE INTO sessions (uuid, name, value, expire_after) VALUES (?, ?, ?, ?)", s.UUID, name, encoded, time.Now().Add(duration))
	if err != nil {
		return err
	}
	s.data[name] = value
	return nil
}

func (s *Session) Get(name string, value any) (err error) {
	if v, ok := s.data[name]; ok {
		switch v := v.(type) {
		case json.RawMessage:
			err := json.Unmarshal(v, value)
			if err != nil {
				return err
			}
			s.data[name] = value
		default:
			x := reflect.ValueOf(v)
			if x.Kind() == reflect.Ptr {
				reflect.ValueOf(value).Set(x)
			} else {
				reflect.ValueOf(value).Elem().Set(x)
			}
		}

		return nil
	}
	var encoded []byte
	err = db.Get(&encoded, "SELECT value FROM sessions WHERE uuid = ? AND name = ?", s.UUID, name)
	if err != nil {
		return err
	}
	err = json.Unmarshal(encoded, value)
	if err != nil {
		return err
	}
	s.data[name] = value
	return nil
}

func (s *Session) Del(name string) error {
	delete(s.data, name)
	_, err := db.Exec("DELETE FROM sessions WHERE uuid = ? AND name = ?", s.UUID, name)
	return err
}

func sessionCleaner() {
	r, err := db.Exec("DELETE FROM sessions WHERE datetime(expire_after) <= CURRENT_TIMESTAMP")
	if err != nil {
		return
	}
	row, _ := r.RowsAffected()
	log.Info("Cleaned ", row, " expired sessions")
}
