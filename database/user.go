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
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql/driver"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/fatih/color"
	"github.com/jmoiron/sqlx"
	"github.com/matthewhartstonge/argon2"
)

type User struct {
	UID      int64       `json:"uid" db:"uid"`
	Username string      `json:"username" db:"username"`
	Email    string      `json:"email" db:"email"`
	Role     string      `json:"role" db:"role"`
	Password string      `json:"-" db:"password"`
	Extra    *User_Extra `json:"extra" db:"extra"`
}

type User_Extra struct {
}

func (e *User_Extra) String() string {
	encoded, _ := json.Marshal(e)
	return string(encoded)
}

func (e *User_Extra) Scan(src any) error {
	switch v := src.(type) {
	case []byte:
		return json.Unmarshal(v, e)
	case string:
		return json.Unmarshal([]byte(v), e)
	default:
		return fmt.Errorf("invalid type")
	}
}

func (e *User_Extra) Value() (driver.Value, error) {
	encoded, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}
	return string(encoded), nil
}

func (user *User) IsAnonymous() bool {
	return user.UID == 1
}

func (user *User) IsAdmin() bool {
	return user.Role == "admin"
}

func (user *User) Create(setuid bool) (err error) {
	var result *sqlx.Row
	if setuid {
		result = db.QueryRowx(`INSERT INTO users (uid, username, email, role, password, extra) VALUES (? ,?, ?, ?, ?, "{}") RETURNING *`, user.UID, user.Username, user.Email, user.Role, user.Password)
	} else {
		result = db.QueryRowx(`INSERT INTO users (username, email, role, password, extra) VALUES (?, ?, ?, ?, "{}") RETURNING *`, user.Username, user.Email, user.Role, user.Password)
	}
	err = result.StructScan(user)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Info(color.YellowString(`用户`), color.MagentaString("[%d] ", user.UID), color.CyanString(user.Username), color.BlueString("[%s]", user.Email), color.YellowString(" 添加成功"))
	return nil
}

func (user *User) Update() error {
	_, err := db.Exec("UPDATE users SET username = ?, email = ?, role = ?, password = ?, extra = ? WHERE uid = ?", user.Username, user.Email, user.Role, user.Password, user.Extra, user.UID)
	return err
}

func (user *User) Delete() error {
	tx, err := db.BeginTxx(context.Background(), nil)
	if err != nil {
		return err
	}
	tx.Exec("DELETE FROM users WHERE uid = ?", user.UID)
	tx.Exec("DELETE FROM pastes WHERE uid = ?", user.UID)
	err = tx.Commit()
	return err
}

func (user *User) SetPassword(password string) error {
	argon := argon2.DefaultConfig()
	encoded, err := argon.HashEncoded([]byte(password))
	if err != nil {
		return err
	}
	user.Password = string(encoded)
	return nil
}

func (user *User) ChangePassword(oldPassword string, newPassword string) error {
	if ok, err := argon2.VerifyEncoded([]byte(oldPassword), []byte(user.Password)); err != nil || !ok {
		return ErrNotFoundOrPasswordWrong
	}
	return user.SetPassword(newPassword)
}

var ErrNotFoundOrPasswordWrong = fmt.Errorf("account not found or bad password")

func Login(account string, password string) (*User, error) {
	result := db.QueryRowx("SELECT uid, username, email, role, password FROM users WHERE email = ? OR username = ?", account, account)
	user := &User{}
	err := result.StructScan(user)
	if err != nil {
		log.Error(err)
		return nil, ErrNotFoundOrPasswordWrong
	}
	if user.Password == "" {
		return nil, ErrNotFoundOrPasswordWrong
	}
	if ok, err := argon2.VerifyEncoded([]byte(password), []byte(user.Password)); err != nil || !ok {
		return nil, ErrNotFoundOrPasswordWrong
	}
	return user, nil
}

func (u *User) Token() string {
	hash := hmac.New(sha256.New, []byte(u.Password))
	buf := [48]byte{}
	binary.Write(bytes.NewBuffer(buf[:0]), binary.BigEndian, u.UID)
	rand.Read(buf[8:16])
	hash.Write(buf[:16])
	copy(buf[16:], hash.Sum(nil))
	return base64.StdEncoding.EncodeToString(buf[:])
}

func GetUserByToken(token string) (*User, error) {
	buf, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, err
	}
	var uid int64
	err = binary.Read(bytes.NewReader(buf[:8]), binary.BigEndian, &uid)
	if err != nil {
		return nil, err
	}
	user, err := GetUser(uid)
	if err != nil {
		return nil, err
	}
	hash := hmac.New(sha256.New, []byte(user.Password))
	hash.Write(buf[:16])
	if !slices.Equal(buf[16:], hash.Sum(nil)) {
		return nil, ErrNotFoundOrPasswordWrong
	}
	return user, nil
}

func GetUser(uid int64) (*User, error) {
	result := db.QueryRowx("SELECT * FROM users WHERE uid = ?", uid)
	user := &User{}
	err := result.StructScan(user)
	if err != nil {
		return nil, ErrNotFoundOrPasswordWrong
	}
	if user.Password == "" {
		return nil, ErrNotFoundOrPasswordWrong
	}
	return user, nil
}
