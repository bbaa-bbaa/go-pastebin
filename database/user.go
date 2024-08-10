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
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"slices"

	"github.com/fatih/color"
	"github.com/matthewhartstonge/argon2"
)

const UserAnonymous = 1

func AddUser(username string, email string, role string, password string) (err error) {
	argon := argon2.DefaultConfig()
	encoded := []byte{}
	if len(password) != 0 {
		encoded, err = argon.HashEncoded([]byte(password))
		if err != nil {
			log.Error(err)
			return err
		}
	}
	_, err = db.Exec(`INSERT INTO users (username, email, role, password, extra) VALUES (?, ?, ?, ?, "{}")`, username, email, role, encoded)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Info(color.YellowString(`用户 `), color.CyanString(username), color.BlueString("["+email+"]"), color.YellowString(" 添加成功"))
	return
}

func AddUID(uid int, username string, email string, role string, password string) (err error) {
	argon := argon2.DefaultConfig()
	encoded := []byte{}
	if len(password) != 0 {
		encoded, err = argon.HashEncoded([]byte(password))
		if err != nil {
			log.Error(err)
			return err
		}
	}
	_, err = db.Exec(`INSERT INTO users (uid, username, email, role, password, extra) VALUES (? ,?, ?, ?, ?, "{}")`, uid, username, email, role, encoded)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Info(color.YellowString(`用户`), color.MagentaString("[%d] ", uid), color.CyanString(username), color.BlueString("[%s]", email), color.YellowString(" 添加成功"))
	return
}

type User struct {
	Uid      int64  `json:"uid"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Password string `json:"-"`
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
	if ok, err := argon2.VerifyEncoded([]byte(password), []byte(user.Password)); err != nil || !ok {
		return nil, ErrNotFoundOrPasswordWrong
	}
	return user, nil
}

func (u *User) Token() string {
	hash := hmac.New(sha256.New, []byte(u.Password))
	buf := [48]byte{}
	binary.Write(bytes.NewBuffer(buf[:0]), binary.BigEndian, u.Uid)
	rand.Read(buf[8:16])
	hash.Write(buf[:16])
	copy(buf[16:], hash.Sum(nil))
	return base64.URLEncoding.EncodeToString(buf[:])
}

func GetUser(token string) (*User, error) {
	buf, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return nil, err
	}
	var uid int64
	err = binary.Read(bytes.NewReader(buf[:8]), binary.BigEndian, &uid)
	if err != nil {
		return nil, err
	}
	result := db.QueryRowx("SELECT uid, username, email, role, password FROM users WHERE uid = ?", uid)
	user := &User{}
	err = result.StructScan(user)
	if err != nil {
		return nil, ErrNotFoundOrPasswordWrong
	}
	hash := hmac.New(sha256.New, []byte(user.Password))
	hash.Write(buf[:16])
	if !slices.Equal(buf[16:], hash.Sum(nil)) {
		return nil, ErrNotFoundOrPasswordWrong
	}
	return user, nil
}
