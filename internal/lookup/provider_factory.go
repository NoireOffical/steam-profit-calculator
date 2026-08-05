package lookup

import "fmt"

const (
	ProviderSteamPublic = "steam_public"
)

// NewClientFromEnvironment builds the configured external lookup client from environment config.
func NewClientFromEnvironment() (Client, error) {
	cfg, err := LoadConfigFromEnvironment()
	if err != nil {
		return nil, err
	}
	return NewClient(cfg)
}

// NewClient builds an external lookup client from provider-aware config.
func NewClient(cfg Config) (Client, error) {
	switch cfg.Provider {
	case "", ProviderSteamPublic:
		return NewSteamPublicClient(cfg)
	default:
		return nil, fmt.Errorf("unsupported lookup provider %q", cfg.Provider)
	}
}
