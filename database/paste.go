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
	"bufio"
	"bytes"
	"crypto/sha256"
	"database/sql"
	"database/sql/driver"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/fatih/color"
	"github.com/gabriel-vasile/mimetype"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/matthewhartstonge/argon2"
	"github.com/mattn/go-sqlite3"
)

type Paste_Hash int64

func (ph Paste_Hash) hex() string {
	return fmt.Sprintf("%016x", uint64(ph))
}

func (ph Paste_Hash) base64() string {
	buf := bytes.NewBuffer([]byte{})
	err := binary.Write(buf, binary.BigEndian, ph)
	if err != nil {
		log.Error(err)
	}
	return base64.URLEncoding.EncodeToString(buf.Bytes())

}

func (ph Paste_Hash) base64WithoutPadding() string {
	return strings.TrimRight(ph.base64(), "=")
}

type Paste struct {
	UUID           string       `db:"uuid"`
	UID            int64        `db:"uid"`
	Content        io.Reader    `db:"-"`
	Hash           Paste_Hash   `db:"hash"`
	Password       string       `db:"password"`
	ExpireAfter    time.Time    `db:"expire_after"`
	AccessCount    int64        `db:"access_count"`
	MaxAccessCount int64        `db:"max_access_count"`
	DeleteIfExpire bool         `db:"delete_if_expire"`
	HoldCount      int64        `db:"hold_count"`
	HoldBefore     time.Time    `db:"hold_before"`
	Extra          *Paste_Extra `db:"extra"`
	CreatedAt      time.Time    `db:"created_at"`
	Short_url      string       `db:"short_url"`
}

func (p *Paste) Base64Hash() string {
	if p.Extra.HashPadding {
		return p.Hash.base64()
	} else {
		return p.Hash.base64WithoutPadding()
	}
}

func (p *Paste) HexHash() string {
	return p.Hash.hex()
}

type Paste_Extra struct {
	MineType    string `json:"mine_type"`
	FileName    string `json:"filename"`
	Size        uint64 `json:"size"`
	HashPadding bool   `json:"hash_padding"`
}

func (e *Paste_Extra) String() string {
	encoded, _ := json.Marshal(e)
	return string(encoded)
}

func (e *Paste_Extra) Scan(src any) error {
	switch v := src.(type) {
	case []byte:
		return json.Unmarshal(v, e)
	case string:
		return json.Unmarshal([]byte(v), e)
	default:
		return fmt.Errorf("invalid type")
	}
}

func (e *Paste_Extra) Value() (driver.Value, error) {
	encoded, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}
	return string(encoded), nil
}

var ErrNoContent = fmt.Errorf("paste no content")
var ErrNotFound = fmt.Errorf("paste not found")
var ErrNoDatabase = fmt.Errorf("database not connect")
var ErrAlreadyExist = fmt.Errorf("paste already exist")
var ErrShortURLAlreadyExist = fmt.Errorf("short url already exist")

const block_size = 1024 * 1024 // 1M
func (p *Paste) Save() (*Paste, error) {
	var paste_file *os.File
	for try := 0; ; {
		paste_uuid, err := uuid.NewV7()
		if err != nil {
			return nil, err
		}
		paste_uuid.Time()
		p.UUID = paste_uuid.String()
		paste_file, err = os.OpenFile(p.Path(), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if os.IsExist(err) {
			if try++; try < 10000 {
				continue
			}
			return nil, err
		}
		break
	}
	p.save(paste_file)
	p.CreatedAt = time.Now()
	p.Extra.HashPadding = p.hashBumpWithShortURL()
	_, err := db.Exec(`INSERT INTO pastes (uuid, hash, password, expire_after, access_count, max_access_count, delete_if_expire, hold_count, hold_before, extra, uid, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.UUID,
		p.Hash,
		p.Password,
		p.ExpireAfter,
		p.AccessCount,
		p.MaxAccessCount,
		p.DeleteIfExpire,
		p.HoldCount,
		p.HoldBefore,
		p.Extra,
		p.UID,
		p.CreatedAt,
	)
	if err != nil {
		if sqliteErr, ok := err.(sqlite3.Error); ok {
			if sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
				os.Remove(paste_file.Name())
				paste, err := QueryPasteByHash(p.Hash)
				if err != nil {
					fmt.Printf("QueryPasteByHash error: %v\n", err)
					return nil, err
				}
				return paste, ErrAlreadyExist
			}
		}
		return nil, err
	}
	log.Info(color.YellowString("Paste "), color.CyanString(p.UUID), color.MagentaString(`[%s]`, p.Extra.FileName), color.YellowString(" 创建成功 "), color.YellowString("Hash: "), color.CyanString(p.HexHash()), color.YellowString(" Size: "), color.CyanString(fmt.Sprint(p.Extra.Size)))
	if p.Short_url == "" {
		p.GenerateShortURL()
	}
	return p, p.CreateShortURL()
}

func (p *Paste) hashBumpWithShortURL() bool {
	count := 0
	err := db.Get(&count, "SELECT COUNT(*) FROM short_url WHERE name = ?", p.Hash.base64WithoutPadding())
	if err != nil {
		return false
	}
	return count != 0
}

func (p *Paste) GenerateShortURL() error {
	sql := "SELECT COALESCE(MAX(LENGTH(name)),0) FROM short_url WHERE "
	alternatives := []any{}
	hash := p.Hash.base64WithoutPadding()
	hash_len := len(hash)
	for i := hash_len - 1; i >= 1; i-- {
		sql += "name = ?"
		if i != 1 {
			sql += " OR "
		}
		alternatives = append(alternatives, hash[i:hash_len])
	}
	var length int
	err := db.Get(&length, sql, alternatives...)
	if err != nil {
		log.Error(err)
		return err
	}
	if length+1 < len(hash) {
		p.Short_url = hash[hash_len-length-1 : hash_len]
	}
	return nil
}

func (p *Paste) SetPassword(password string) (err error) {
	argon := argon2.DefaultConfig()
	encoded := []byte{}
	if len(password) != 0 {
		encoded, err = argon.HashEncoded([]byte(password))
		if err != nil {
			log.Error(err)
			return err
		}
	}
	p.Password = string(encoded)
	return nil
}

func (p *Paste) Update() error {
	if len(p.UUID) == 0 {
		return ErrNotFound
	}
	paste_file, err := os.OpenFile(p.Path(), os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	p.save(paste_file)
	return p.UpdateMetadata()
}

func (p *Paste) Path() string {
	return filepath.Join(GetPastesDir(), p.UUID)
}

func (p *Paste) UpdateMetadata() error {
	p.CreatedAt = time.Now()
	_, err := db.Exec(`UPDATE pastes SET hash = ?, password = ?, expire_after = ?, access_count = ?, max_access_count = ?, delete_if_expire = ?, hold_count = ?, hold_before = ?, extra = ?, created_at = ?, WHERE uuid = ?`,
		p.Hash,
		p.Password,
		p.ExpireAfter,
		p.AccessCount,
		p.MaxAccessCount,
		p.DeleteIfExpire,
		p.HoldCount,
		p.HoldBefore,
		p.Extra,
		p.UUID,
		p.CreatedAt,
	)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Info(color.YellowString("Paste "), color.CyanString(p.UUID), color.MagentaString(`[%s]`, p.Extra.FileName), color.YellowString(" 更新成功 "), color.YellowString("Hash: "), color.CyanString(p.HexHash()), color.YellowString(" Size: "), color.CyanString(fmt.Sprint(p.Extra.Size)))
	return nil
}

func (p *Paste) Hold() error {
	_, err := db.Exec(`UPDATE pastes SET hold_count = hold_count + 1 WHERE uuid = ?`,
		p.UUID,
	)
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (p *Paste) Unhold() error {
	_, err := db.Exec(`UPDATE pastes SET hold_count = hold_count - 1 WHERE uuid = ? AND hold_count > 0`,
		p.UUID,
	)
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (p *Paste) Access(hold_before time.Time) error {
	_, err := db.Exec(`UPDATE pastes SET access_count = access_count + 1, hold_before = ? WHERE uuid = ?`,
		hold_before,
		p.UUID,
	)
	if err != nil {
		log.Error(err)
		return err
	}
	p.AccessCount++
	return nil
}

func (p *Paste) Delete() error {
	_, err := db.Exec(`DELETE FROM pastes WHERE uuid = ?`, p.UUID)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Info(color.YellowString("Paste "), color.CyanString(p.UUID), color.RedString(" 删除"))
	return nil
}

func (p *Paste) save(paste_file *os.File) {
	reader := bufio.NewReader(p.Content)
	hash := xxhash.New()
	buf := make([]byte, block_size)
	var mine_detector *io.PipeWriter
	var mine_result chan string
	mine_detect_complete_flag := true
	if p.Extra.MineType == "" || true {
		mine_detect_complete_flag = false
		mine_detector, mine_result = p.mineTypeDetector()
	}

	for {
		n, err := reader.Read(buf)
		if err != nil {
			break
		}
		hash.Write(buf[:n])
		paste_file.Write(buf[:n])
		if !mine_detect_complete_flag {
			_, err := mine_detector.Write(buf[:n])
			if err != nil {
				mine_detect_complete_flag = true
			}
		}
		p.Extra.Size += uint64(n)
	}
	if p.Extra.MineType == "" {
		mine_detector.Close()
		p.Extra.MineType = <-mine_result
	}
	paste_file.Close()
	if r, ok := p.Content.(io.ReadCloser); ok {
		r.Close()
	}
	hash_buf := make([]byte, 8)
	binary.BigEndian.PutUint64(hash_buf, hash.Sum64())
	binary.Read(bytes.NewReader(hash_buf), binary.BigEndian, &p.Hash)
}

func (p *Paste) mineTypeDetector() (w *io.PipeWriter, result chan string) {
	r, w := io.Pipe()
	result = make(chan string, 1)
	go func() {
		mtype, err := mimetype.DetectReader(r)
		r.Close()
		if err != nil {
			log.Error(err)
			result <- ""
		}
		result <- mtype.String()
	}()
	return w, result
}

var ShortURLRule = regexp.MustCompile(`^[a-zA-Z0-9_\.-]+$`)

func (p *Paste) UpdateShortURL() error {
	if p.Short_url == "" {
		return ErrShortURLAlreadyExist
	}

	if !ShortURLRule.MatchString(p.Short_url) {
		return ErrShortURLAlreadyExist
	}

	result, err := db.Exec(`UPDATE short_url SET name = ? WHERE target = ?`, p.Short_url, p.UUID)
	if err != nil {
		if sqliteErr, ok := err.(sqlite3.Error); ok {
			if sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
				return ErrShortURLAlreadyExist
			}
		}
		log.Error(err)
		return err
	}
	if rowsAffected, err := result.RowsAffected(); err != nil {
		log.Error(err)
		return err
	} else if rowsAffected != 0 {
		return nil
	}
	return p.CreateShortURL()
}

func (p *Paste) CreateShortURL() error {
	if p.Short_url == "" {
		return ErrShortURLAlreadyExist
	}

	if !ShortURLRule.MatchString(p.Short_url) {
		return ErrShortURLAlreadyExist
	}

	_, err := db.Exec(`INSERT INTO short_url (name,target) VALUES (?, ?)`, p.Short_url, p.UUID)
	if err != nil {
		if sqliteErr, ok := err.(sqlite3.Error); ok {
			if sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
				return ErrShortURLAlreadyExist
			}
		}
		return err
	}
	log.Info(color.YellowString("Paste "), color.CyanString(p.Short_url), color.YellowString(" -> "), color.CyanString(p.UUID), color.YellowString(" 短链接创建成功"))
	return nil
}

func (p *Paste) Token(ExpireAfter time.Time) string {
	buf := [40]byte{}
	binary.Write(bytes.NewBuffer(buf[:0]), binary.BigEndian, ExpireAfter.UnixMilli())
	hash := sha256.New()
	hash.Write(buf[:8])
	paste_uuid, _ := uuid.Parse(p.UUID)
	buuid, _ := paste_uuid.MarshalBinary()
	hash.Write(buuid)
	copy(buf[8:], hash.Sum([]byte{}))
	return base64.URLEncoding.EncodeToString(buf[:])
}

func (p *Paste) VerifyToken(token string) bool {
	buf, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return false
	}
	if len(buf) != 40 {
		return false
	}
	expire := time.UnixMilli(int64(binary.BigEndian.Uint64(buf[:8])))
	if expire.Before(time.Now()) {
		return false
	}
	hash := sha256.New()
	hash.Write(buf[:8])
	paste_uuid, _ := uuid.Parse(p.UUID)
	buuid, _ := paste_uuid.MarshalBinary()
	hash.Write(buuid)
	return bytes.Equal(hash.Sum([]byte{}), buf[8:])
}

func QueryPasteByHash(hash Paste_Hash) (*Paste, error) {
	row := db.QueryRowx(`SELECT p.*, COALESCE(s.name,"") AS short_url FROM pastes p LEFT JOIN short_url s ON s.target=p.uuid WHERE hash = ?`, hash)
	return parsePaste(row)
}

func QueryPasteByUUID(uuid string) (*Paste, error) {
	row := db.QueryRowx(`SELECT p.*, COALESCE(s.name,"") AS short_url FROM pastes p LEFT JOIN short_url s ON s.target=p.uuid WHERE p.uuid = ?`, uuid)
	return parsePaste(row)
}

func parsePaste(row *sqlx.Row) (*Paste, error) {
	paste := &Paste{}
	err := row.StructScan(paste)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		log.Error(err)
		return nil, err
	}
	return paste, nil
}

func QueryPasteByShortURLOrHash(name string) (*Paste, error) {
	var hash int64
	hash_query := false
	if len(name) == 11 {
		hash_base64 := name + "="
		hash_buf, err := base64.URLEncoding.DecodeString(hash_base64)
		if err == nil {
			err := binary.Read(bytes.NewReader(hash_buf), binary.BigEndian, &hash)
			if err == nil {
				hash_query = true
			}
		}
	}
	if !hash_query {
		row := db.QueryRowx(`SELECT p.*, s.name AS short_url FROM pastes p INNER JOIN short_url s ON p.uuid = s.target WHERE s.name = ?`, name)
		return parsePaste(row)
	} else {
		row := db.QueryRowx(`SELECT p.*, s.name AS short_url FROM pastes p LEFT JOIN short_url s ON p.uuid = s.target WHERE s.name = ? OR p.hash = ?`, name, hash)
		return parsePaste(row)
	}
}
