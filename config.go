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

package pastebin

import (
	"os"

	"cgit.bbaa.fun/bbaa/go-pastebin/database"
	"gopkg.in/yaml.v3"
)

type Pastebin_Config struct {
	SiteName string `yaml:"site_name"`
}

var Config *Pastebin_Config = &Pastebin_Config{
	SiteName: "Pastebin",
}

func SaveConfig() {
	config, _ := yaml.Marshal(Config)
	os.WriteFile(database.GetConfigPath(), config, 0644)
}

func LoadConfig() {
	config_file, err := os.ReadFile(database.GetConfigPath())
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
