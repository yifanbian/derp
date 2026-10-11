package main

import (
	"context"
	"fmt"
	"time"

	tailscale "tailscale.com/client/tailscale/v2"
	"tailscale.com/types/key"
	"tailscale.com/util/set"
)

// Least privilege scope required to list devices.
const scopeDevicesCoreRead = "devices:core:read"

func newTailscaleClient(t Tailnet) (*tailscale.Client, error) {
	c := &tailscale.Client{Tailnet: t.Name}

	switch a := t.Auth; {
	case a.ClientCredential != nil:
		c.Auth = &tailscale.OAuth{
			ClientID:     a.ClientCredential.ID,
			ClientSecret: a.ClientCredential.Secret,
			Scopes:       []string{scopeDevicesCoreRead},
		}
	case a.IdentityFederation != nil && a.IdentityFederation.AzureIMDS != nil:
		imdsClientID := a.IdentityFederation.AzureIMDS.ClientID
		c.Auth = &tailscale.IdentityFederation{
			ClientID: a.IdentityFederation.ID,
			IDTokenFunc: func() (string, error) {
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()
				return azureIMDSToken(ctx, imdsClientID)
			},
		}
	default:
		return nil, fmt.Errorf("tailnet %q: unsupported auth", t.Name)
	}
	return c, nil
}

// fetchNodeKeys lists the tailnet's devices and returns their node keys.
func fetchNodeKeys(ctx context.Context, c *tailscale.Client) (set.Set[key.NodePublic], error) {
	devices, err := c.Devices().List(ctx)
	if err != nil {
		return nil, err
	}
	keys := make(set.Set[key.NodePublic], len(devices))
	for _, d := range devices {
		if d.NodeKey == "" {
			continue
		}
		var k key.NodePublic
		if err := k.UnmarshalText([]byte(d.NodeKey)); err != nil {
			return nil, fmt.Errorf("device %s: invalid node key: %w", d.NodeID, err)
		}
		keys.Add(k)
	}
	return keys, nil
}
