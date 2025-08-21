// Copyright 2022 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"fmt"
	"log/slog"
	"os"
	"sync"

	"go.yaml.in/yaml/v3"
)

type Config struct {
	Auth              *AuthConfig `yaml:"auth"`
	IncludedWorkspace []string    `yaml:"included_workspaces"`
}

type Handler struct {
	sync.RWMutex
	Config *Config
}

func (ch *Handler) GetConfig() *Config {
	ch.RLock()
	defer ch.RUnlock()
	return ch.Config
}

func (ch *Handler) ReloadConfig(f string, logger *slog.Logger) error {
	config := &Config{}
	var err error

	yamlReader, err := os.Open(f)
	if err != nil {
		return fmt.Errorf("error opening config file %q: %s", f, err)
	}
	defer yamlReader.Close()
	decoder := yaml.NewDecoder(yamlReader)
	decoder.KnownFields(true)

	if err = decoder.Decode(config); err != nil {
		return fmt.Errorf("error parsing config file %q: %s", f, err)
	}

	ch.Lock()
	ch.Config = config
	ch.Unlock()
	return nil
}
