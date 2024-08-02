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
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
)

var base_path = "/var/lib/go-pastebin"

func GetDBPath() string {
	return filepath.Join(base_path, "go-pastebin.db")
}

func GetPastesDir() string {
	return filepath.Join(base_path, "pastes")
}

func GetConfigPath() string {
	return filepath.Join(base_path, "config.yaml")
}

func ensureDir(sub_path string) error {
	err := os.MkdirAll(filepath.Join(base_path, sub_path), 0755)
	if err != nil && !errors.Is(err, os.ErrExist) {
		if errors.Is(err, os.ErrPermission) && strings.HasPrefix(base_path, "/var") {
			return fallbackBaseDir(sub_path)
		}
		return err
	}
	if errors.Is(err, os.ErrExist) {
		tempfile, err := os.CreateTemp(filepath.Join(base_path, sub_path), "test")
		if err != nil && strings.HasPrefix(base_path, "/var") {
			return fallbackBaseDir(sub_path)
		}
		tempfile.Close()
		os.Remove(tempfile.Name())
	}
	return nil
}

func fallbackBaseDir(sub_path string) error {
	log.Warning("数据库路径: ", color.BlueString(base_path), " 无权限访问")
	workdir, err := os.Getwd()
	if err != nil {
		return err
	}
	base_path = filepath.Join(workdir, "data")
	return ensureDir(sub_path)
}
