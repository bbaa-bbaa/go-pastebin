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
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

type Pastebin_Config struct {
	SiteName            string `yaml:"site_name"`
	SupportNoFilename   bool   `yaml:"support_no_filename"`
	Mode                string `yaml:"mode"`
	AllowHTML           bool   `yaml:"allow_html"`
	AllowAnonymous      bool   `yaml:"allow_anonymous"`
	UserCookieMaxAge    int    `yaml:"user_cookie_max_age"`
	PasteAssessTokenAge int    `yaml:"paste_assess_token_age"`
	DataDir             string `yaml:"data_dir"`
}

var Config *Pastebin_Config = &Pastebin_Config{
	SiteName:            "Pastebin",
	SupportNoFilename:   true,
	Mode:                "release",
	AllowHTML:           false,
	AllowAnonymous:      true,
	UserCookieMaxAge:    86400 * 30,
	PasteAssessTokenAge: 86400,
	DataDir:             "/var/lib/go-pastebin",
}

func SaveConfig() {
	config, _ := yaml.Marshal(Config)
	os.WriteFile(GetConfigPath(), config, 0644)
}

func LoadConfig() {
	if runtime.GOOS == "windows" {
		workdir, err := os.Getwd()
		if err != nil {
			workdir = "."
		}
		Config.DataDir = filepath.Join(workdir, "data")
	}
	config_file, err := os.ReadFile(GetConfigPath())
	if err != nil {
		SaveConfig()
		return
	}
	var config Pastebin_Config
	err = yaml.Unmarshal(config_file, &config)
	if err != nil {
		SaveConfig()
		return
	}
	Config = &config
}
