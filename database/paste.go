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

var ReservedURL = regexp.MustCompile(`^(sw\.js(\.map)?|workbox.*?\.js(\.map)?|manifest\.json|favicon\.ico|robots\.txt|index\.(x|s)?htm(l)?|admin\.(x|s)?htm(l)?)$`)

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
	MimeType    string `json:"mime_type"`
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
var ErrInvalidShortURL = fmt.Errorf("invalid short url")

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
		} else if err != nil {
			return nil, err
		}
		break
	}
	err := p.save(paste_file)
	if err != nil {
		os.Remove(paste_file.Name())
		return nil, err
	}
	p.CreatedAt = time.Now()
	p.Extra.HashPadding = ShortURLExist(p.Hash.base64())
	_, err = db.Exec(`INSERT INTO pastes (uuid, hash, password, expire_after, access_count, max_access_count, delete_if_expire, hold_count, hold_before, extra, uid, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
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

func ShortURLExist(name string) bool {
	count := 0
	err := db.Get(&count, "SELECT COUNT(*) FROM short_url WHERE name = ?", name)
	if err != nil {
		return false
	}
	return count != 0
}

func HashExist(hash any) bool {
	var decoded_hash Paste_Hash
	var err error
	switch query_hash := hash.(type) {
	case int64:
		decoded_hash = Paste_Hash(query_hash)
	case string:
		decoded_hash, err = DecodeBase64Hash(query_hash)
		if err != nil {
			return false
		}
	default:
		return false
	}
	count := 0
	err = db.Get(&count, "SELECT COUNT(*) FROM pastes WHERE hash = ?", decoded_hash)
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

func (p *Paste) Update() (paste *Paste, err error) {
	if len(p.UUID) == 0 {
		return p, ErrNotFound
	}
	var paste_file *os.File
	if p.Content != nil {
		paste_file, err = os.CreateTemp(GetPastesDir(), "paste_update")
		if err != nil {
			return p, err
		}
		p.save(paste_file)
	}
	err = p.UpdateMetadata()
	if err != nil {
		if sqliteErr, ok := err.(sqlite3.Error); ok {
			if sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
				if paste_file != nil {
					os.Remove(paste_file.Name())
				}
				paste, err := QueryPasteByHash(p.Hash)
				if err != nil {
					return nil, err
				}
				return paste, ErrAlreadyExist
			}
		}
		return nil, err
	}
	if paste_file != nil {
		os.Remove(p.Path())
		os.Rename(paste_file.Name(), p.Path())
	}
	return p, p.UpdateShortURL()
}

func (p *Paste) Path() string {
	return filepath.Join(GetPastesDir(), p.UUID)
}

func (p *Paste) UpdateMetadata() error {
	p.CreatedAt = time.Now()
	_, err := db.Exec(`UPDATE pastes SET hash = ?, password = ?, expire_after = ?, access_count = ?, max_access_count = ?, delete_if_expire = ?, hold_count = ?, hold_before = ?, extra = ?, created_at = ? WHERE uuid = ?`,
		p.Hash,
		p.Password,
		p.ExpireAfter,
		p.AccessCount,
		p.MaxAccessCount,
		p.DeleteIfExpire,
		p.HoldCount,
		p.HoldBefore,
		p.Extra,
		p.CreatedAt,
		p.UUID,
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

var ErrPasteHold = fmt.Errorf("paste hold")

func (p *Paste) delete(force bool) error {
	if !force && (p.HoldCount > 0 || p.HoldBefore.After(time.Now())) {
		return ErrPasteHold
	}
	_, err := db.Exec(`DELETE FROM pastes WHERE uuid = ?`, p.UUID)
	if err != nil {
		log.Error(err)
		return err
	}
	os.Remove(p.Path())
	log.Info(color.YellowString("Paste "), color.CyanString(p.UUID), color.RedString(" 删除"))
	return nil
}

func (p *Paste) Delete() error {
	return p.delete(false)
}

func (p *Paste) ForceDelete() error {
	return p.delete(true)
}

func (p *Paste) FlagDelete() error {
	_, err := db.Exec(`UPDATE pastes SET expire_after = datetime("now"), max_access_count = -1, delete_if_expire = 1 WHERE uuid = ?`, p.UUID)
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (p *Paste) save(paste_file *os.File) error {
	reader := bufio.NewReader(p.Content)
	hash := xxhash.New()
	buf := make([]byte, block_size)
	var mime_detector *io.PipeWriter
	var mime_result chan string
	mime_detect_complete_flag := true
	if p.Extra.MimeType == "" || strings.HasPrefix(p.Extra.MimeType, "text/") && !strings.Contains(p.Extra.MimeType, "charset=") {
		mime_detect_complete_flag = false
		mime_detector, mime_result = p.mimeTypeDetector(p.Extra.MimeType)
	}
	p.Extra.Size = 0
	for {
		n, err := reader.Read(buf)
		if err != nil {
			break
		}
		hash.Write(buf[:n])
		paste_file.Write(buf[:n])
		if !mime_detect_complete_flag {
			_, err := mime_detector.Write(buf[:n])
			if err != nil {
				mime_detect_complete_flag = true
			}
		}
		p.Extra.Size += uint64(n)
	}
	if mime_detector != nil {
		mime_detector.Close()
		p.Extra.MimeType = <-mime_result
	}
	paste_file.Close()
	if r, ok := p.Content.(io.ReadCloser); ok {
		r.Close()
	}
	hash_buf := make([]byte, 8)
	binary.BigEndian.PutUint64(hash_buf, hash.Sum64())
	binary.Read(bytes.NewReader(hash_buf), binary.BigEndian, &p.Hash)
	return nil
}

func (p *Paste) mimeTypeDetector(fallback string) (w *io.PipeWriter, result chan string) {
	r, w := io.Pipe()
	result = make(chan string, 1)
	go func() {
		mtype, err := mimetype.DetectReader(r)
		r.Close()
		if err != nil {
			result <- fallback
			return
		}
		result <- mtype.String()
	}()
	return w, result
}

var ShortURLRule = regexp.MustCompile(`^[a-zA-Z0-9_\.-]+$`)

func CheckShortURL(p *Paste) error {
	if p.Short_url == "" {
		return ErrInvalidShortURL
	}
	if !ShortURLRule.MatchString(p.Short_url) {
		return ErrInvalidShortURL
	}
	if ReservedURL.MatchString(p.Short_url) {
		return ErrShortURLAlreadyExist
	}
	if HashExist(p.Short_url) {
		return ErrShortURLAlreadyExist
	}
	return nil
}

func (p *Paste) UpdateShortURL() error {
	if err := CheckShortURL(p); err != nil {
		return err
	}

	result, err := db.Exec(`UPDATE short_url SET name = ? WHERE target = ?`, p.Short_url, p.UUID)
	if err != nil {
		if sqliteErr, ok := err.(sqlite3.Error); ok {
			if sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
				return ErrShortURLAlreadyExist
			}
		}
		return err
	}
	if rowsAffected, err := result.RowsAffected(); err != nil {
		return err
	} else if rowsAffected != 0 {
		return nil
	}
	return p.CreateShortURL()
}

func (p *Paste) CreateShortURL() error {
	if err := CheckShortURL(p); err != nil {
		return err
	}

	_, err := db.Exec(`INSERT INTO short_url (name,target) VALUES (?, ?)`, p.Short_url, p.UUID)
	if err != nil {
		if sqliteErr, ok := err.(sqlite3.Error); ok {
			if sqliteErr.Code == sqlite3.ErrConstraint {
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

func (p *Paste) Valid() bool {
	return (p.MaxAccessCount == 0 || p.AccessCount < p.MaxAccessCount) && (p.ExpireAfter.IsZero() || p.ExpireAfter.After(time.Now()))
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
	if !paste.Valid() && paste.DeleteIfExpire {
		err := paste.Delete()
		if err == nil {
			return nil, ErrNotFound
		}
	}
	return paste, nil
}

var ErrDecodeBase64Hash = fmt.Errorf("invalid base64 hash")

func DecodeBase64Hash(base64_hash string) (Paste_Hash, error) {
	base64_hash = strings.TrimRight(base64_hash, "=")
	padding := (4 - len(base64_hash)%4) % 4
	if padding == 3 {
		return 0, ErrDecodeBase64Hash
	}
	if padding <= 2 {
		base64_hash += strings.Repeat("=", padding)
	}
	hash_buf, err := base64.URLEncoding.DecodeString(base64_hash)
	if err != nil {
		return 0, err
	}
	if len(hash_buf) != 8 {
		return 0, ErrDecodeBase64Hash
	}
	var hash int64
	err = binary.Read(bytes.NewReader(hash_buf), binary.BigEndian, &hash)
	if err != nil {
		return 0, err
	}
	return Paste_Hash(hash), nil
}

func QueryPasteByShortURLOrHash(name string) (p *Paste, err error) {
	var hash Paste_Hash
	hash_query := false
	if hash, err = DecodeBase64Hash(name); err == nil {
		hash_query = true
	}
	if !hash_query {
		row := db.QueryRowx(`SELECT p.*, COALESCE(s.name,"") AS short_url FROM pastes p INNER JOIN short_url s ON p.uuid = s.target WHERE s.name = ?`, name)
		return parsePaste(row)
	} else {
		row := db.QueryRowx(`SELECT p.*, COALESCE(s.name,"") AS short_url FROM pastes p LEFT JOIN short_url s ON p.uuid = s.target WHERE s.name = ? OR p.hash = ?`, name, hash)
		return parsePaste(row)
	}
}

func ResetHoldCount() error {
	_, err := db.Exec(`UPDATE pastes SET hold_count = 0 WHERE hold_count > 0`)
	if err != nil {
		log.Error(err)
	}
	return err
}

func QueryAllPasteByUser(uid int64, page int64, page_size int64) (pastes []*Paste, total int, err error) {
	err = db.Get(&total, `SELECT COUNT(*) FROM pastes WHERE uid = ?`, uid)
	if err != nil {
		log.Error(err)
		return nil, 0, err
	}
	index := max(page-1, 0) * page_size
	rows, err := db.Queryx(`SELECT p.*, COALESCE(s.name,"") AS short_url FROM pastes p LEFT JOIN short_url s ON p.uuid = s.target WHERE uid = ? ORDER BY p.uuid ASC LIMIT ? OFFSET ?`, uid, page_size, index)
	if err != nil {
		log.Error(err)
		return nil, 0, err
	}
	defer rows.Close()
	for rows.Next() {
		paste := &Paste{}
		err := rows.StructScan(paste)
		if err != nil {
			log.Error(err)
			continue
		}
		pastes = append(pastes, paste)
	}
	return
}

func pasteCleaner() {
	rows, err := db.Queryx(`DELETE FROM pastes WHERE (expire_after < datetime("now") AND delete_if_expire = 1 OR max_access_count > 0 AND access_count >= max_access_count) AND hold_count = 0 AND hold_before < datetime("now") RETURNING uuid`)
	if err != nil {
		log.Error(err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		err := rows.Err()
		if err != nil {
			log.Error(err)
			return
		}
		var uuid string
		err = rows.Scan(&uuid)
		if err != nil {
			log.Error(err)
			continue
		}
		log.Info(color.YellowString("清理过期 Paste:"), color.CyanString(uuid))
	}
}
