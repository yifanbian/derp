package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Listen          string        `yaml:"listen"`
	RefreshInterval time.Duration `yaml:"refreshInterval"`
	Tailnets        []Tailnet     `yaml:"tailnets"`
}

type Tailnet struct {
	Name string      `yaml:"name"`
	Auth TailnetAuth `yaml:"auth"`
}

type TailnetAuth struct {
	ClientCredential   *ClientCredential   `yaml:"clientCredential"`
	IdentityFederation *IdentityFederation `yaml:"identityFederation"`
}

type ClientCredential struct {
	ID     string `yaml:"id"`
	Secret string `yaml:"secret"`
}

type IdentityFederation struct {
	ID        string     `yaml:"id"`
	AzureIMDS *AzureIMDS `yaml:"azureImds"`
}

type AzureIMDS struct {
	ClientID string `yaml:"clientId"`
}

func LoadConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if c.Listen == "" {
		c.Listen = ":8080"
	}
	if c.RefreshInterval == 0 {
		c.RefreshInterval = 5 * time.Minute
	}
	if err := c.validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Config) validate() error {
	if c.RefreshInterval < time.Second {
		return errors.New("refreshInterval must be at least 1s")
	}
	if len(c.Tailnets) == 0 {
		return errors.New("no tailnets configured")
	}
	seen := map[string]bool{}
	for i, t := range c.Tailnets {
		if t.Name == "" {
			return fmt.Errorf("tailnets[%d]: name is required", i)
		}
		if seen[t.Name] {
			return fmt.Errorf("tailnets[%d]: duplicate name %q", i, t.Name)
		}
		seen[t.Name] = true

		a := t.Auth
		switch {
		case a.ClientCredential != nil && a.IdentityFederation != nil:
			return fmt.Errorf("tailnet %q: set only one of clientCredential and identityFederation", t.Name)
		case a.ClientCredential != nil:
			if a.ClientCredential.ID == "" || a.ClientCredential.Secret == "" {
				return fmt.Errorf("tailnet %q: clientCredential requires id and secret", t.Name)
			}
		case a.IdentityFederation != nil:
			if a.IdentityFederation.ID == "" {
				return fmt.Errorf("tailnet %q: identityFederation requires id", t.Name)
			}
			if a.IdentityFederation.AzureIMDS == nil {
				return fmt.Errorf("tailnet %q: identityFederation requires azureImds", t.Name)
			}
		default:
			return fmt.Errorf("tailnet %q: auth is required", t.Name)
		}
	}
	return nil
}
