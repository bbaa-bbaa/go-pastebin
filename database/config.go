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
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"runtime"

	"github.com/go-webauthn/webauthn/webauthn"
	"gopkg.in/yaml.v3"
)

type Pastebin_Config struct {
	SiteName            string           `yaml:"site_name"`
	SiteTitle           string           `yaml:"site_title"`
	SiteDomain          string           `yaml:"site_domain"`
	WebauthnOrigins     []string         `yaml:"webauthn_origins"`
	SupportNoFilename   bool             `yaml:"support_no_filename"`
	Mode                string           `yaml:"mode"`
	AllowHTML           bool             `yaml:"allow_html"`
	AllowAnonymous      bool             `yaml:"allow_anonymous"`
	UserCookieMaxAge    int              `yaml:"user_cookie_max_age"`
	PasteAssessTokenAge int              `yaml:"paste_assess_token_age"`
	CustomTemplateDir   string           `yaml:"custom_template_dir"`
	SessionCookie       string           `yaml:"session_cookie"`
	dataDir             *string          `yaml:"-"`
	webauthnConfig      *webauthn.Config `yaml:"-"`
}

var Config *Pastebin_Config = &Pastebin_Config{
	SiteName:            "Pastebin",
	SiteTitle:           "",
	SiteDomain:          "go-pastebin.app",
	WebauthnOrigins:     []string{"https://go-pastebin.app"},
	SupportNoFilename:   true,
	Mode:                "release",
	AllowHTML:           false,
	AllowAnonymous:      true,
	UserCookieMaxAge:    86400 * 30,
	PasteAssessTokenAge: 86400,
	SessionCookie:       "PASTEBIN_SESSION",
	CustomTemplateDir:   "",
	dataDir:             flag.String("data", "/var/lib/go-pastebin", "Data directory"),
	webauthnConfig:      &webauthn.Config{},
}

var HTMLTAGRegex = regexp.MustCompile(`(?i)<[^>]*>`)

func SaveConfig() {
	config, _ := yaml.Marshal(Config)
	os.WriteFile(GetConfigPath(), config, 0644)
}

func LoadConfig() {
	flag.Parse()
	if runtime.GOOS == "windows" {
		workdir, err := os.Getwd()
		if err != nil {
			workdir = "."
		}
		*Config.dataDir = filepath.Join(workdir, "data")
	}
	ensureDir("pastes")
	config_file, err := os.ReadFile(GetConfigPath())
	if err != nil {
		SaveConfig()
		return
	}
	yaml.Unmarshal(config_file, &Config)
	if Config.SiteTitle == "" {
		Config.SiteTitle = Config.SiteName
	}
	Config.webauthnConfig.RPID = Config.SiteDomain
	Config.webauthnConfig.RPDisplayName = Config.SiteTitle
	Config.webauthnConfig.RPOrigins = Config.WebauthnOrigins
	SaveConfig()
}
