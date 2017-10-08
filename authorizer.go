/*
Copyright 2017 Home Office All rights reserved.

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

package main

import (
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	core "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/security/podsecuritypolicy"
)

// podAuthorizer is used to wrap the interaction with the psp runtime
type podAuthorizer struct {
	// the configuration for the enforcer
	config *podAuthorizerConfig
	// the enforcement providers
	providers map[string]podsecuritypolicy.Provider
}

// podAuthorizerConfig the security policies configuration
type podAuthorizerConfig struct {
	// Defaul is the name of the default policy to user
	Default string `yaml:"default"`
	// IgnoreNamespaces is a collection of namespace to bypass enforcement
	IgnoreNamespaces []string `yaml:"ignoreNamespaces"`
	// Policies is a list of pod security policies which are available
	Policies map[string]extensions.PodSecurityPolicySpec `yaml:"policies"`
	// NamespaceMapping is a predefined list of namespace to policy mapping
	NamespaceMapping map[string]string `yaml:"namespaceMapping"`
}

// authorize is responsible for adding a policy to the enforcers
func (c *podAuthorizer) authorize(policy string, pod *core.Pod) (bool, field.ErrorList) {
	name := c.config.Default

	// @check is this namespace being ignored
	for _, x := range c.config.IgnoreNamespaces {
		if pod.Namespace == x {
			log.WithFields(log.Fields{
				"namespace": pod.Namespace,
				"pod":       pod.GenerateName,
			}).Info("ignoring authorization on this namespace")

			return true, field.ErrorList{}
		}
	}

	// @step: select the policy to apply against
	// - first we look for a default mappings
	if override, found := c.config.defaultNamespaceMapping(pod.Namespace); found {
		name = override
	}
	// - then we check for a user override
	if policy != "" {
		name = policy
	}

	// @check the policy exists
	provider, found := c.providers[name]
	if !found {
		return false, field.ErrorList{{Detail: "no such policy found", Type: field.ErrorTypeNotFound}}
	}

	// @check for violation of pod policy
	violations := provider.ValidatePodSecurityContext(pod, field.NewPath("", ""))
	if len(violations) > 0 {
		return false, violations
	}

	// @check if of the container security policies
	for _, container := range pod.Spec.Containers {
		if container.SecurityContext == nil {
			continue
		}
		violations = provider.ValidateContainerSecurityContext(pod, &container, field.NewPath("", ""))
		if len(violations) > 0 {
			return false, violations
		}
	}

	return true, field.ErrorList{}
}

// newPodAuthorizer creates and returns a pod authorization implementation
func newPodAuthorizer(config *podAuthorizerConfig) (*podAuthorizer, error) {
	if err := config.isValid(); err != nil {
		return nil, err
	}

	providers := make(map[string]podsecuritypolicy.Provider, 0)
	for name, policy := range config.Policies {
		pv, err := podsecuritypolicy.NewSimpleProvider(&extensions.PodSecurityPolicy{
			ObjectMeta: metav1.ObjectMeta{Name: name}, Spec: policy}, "", podsecuritypolicy.NewSimpleStrategyFactory())
		if err != nil {
			return nil, fmt.Errorf("unable to load policy '%s', error: '%q'", name, err)
		}

		providers[name] = pv
	}

	return &podAuthorizer{
		config:    config,
		providers: providers,
	}, nil
}

// isValid checks if the pod authorization config is valid
func (c *podAuthorizerConfig) isValid() error {
	if len(c.Policies) == 0 {
		return errors.New("zero policies defined")
	}

	if c.Default == "" {
		return errors.New("no default security policy defined")
	}

	if _, found := c.Policies[c.Default]; !found {
		return errors.New("the default policy not found in policies")
	}

	if c.NamespaceMapping != nil {
		for namespace, name := range c.NamespaceMapping {
			if _, found := c.Policies[name]; !found {
				return fmt.Errorf("the mapping for namespace: '%q' policy: '%q' does not exist", namespace, name)
			}
		}
	}

	return nil
}

// defaultNamespaceMapping returns a
func (c *podAuthorizerConfig) defaultNamespaceMapping(namespace string) (string, bool) {
	if c.NamespaceMapping == nil {
		return "", false
	}
	name, found := c.NamespaceMapping[namespace]

	return name, found
}
