/*
Copyright 2018 Home Office All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package scripts

import (
	"time"

	"github.com/UKHomeOffice/policy-admission/pkg/api"
)

const (
	// Name is the name of the authorizer
	Name = "scripts"
)

// Config is the configuration for the authorizer
type Config struct {
	// IgnoreNamespaces is list of namespace to
	IgnoreNamespaces []string `yaml:"ignored-namespaces" json:"ignored-namespaces"`
	// FilterOn is the kind of object you wish to filter is any
	FilterOn string `yaml:"filter-on" json:"filter-on"`
	// Script is the javascript to run on the object
	Script string `yaml:"script" json:"script"`
	// Timeout the timeout applied to the script
	Timeout time.Duration `yaml:"timeout" json:"timeout"`
}

// NewDefaultConfig is the default configuration
func NewDefaultConfig() *Config {
	return &Config{
		IgnoreNamespaces: []string{"kube-system", "kube-public", "kube-admission"},
		FilterOn:         api.FilterAll,
		Timeout:          5 * time.Second,
	}
}
