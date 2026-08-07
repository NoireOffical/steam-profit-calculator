package main

import "testing"

func TestServerAddressFromEnvironment(t *testing.T) {
	t.Setenv("STEAM_PROFIT_SERVER_ADDR", "")
	t.Setenv("PORT", "")
	address, err := serverAddressFromEnvironment()
	if err != nil || address != defaultAddress {
		t.Fatalf("default address = %q / %v", address, err)
	}

	t.Setenv("PORT", "12345")
	address, err = serverAddressFromEnvironment()
	if err != nil || address != "0.0.0.0:12345" {
		t.Fatalf("platform address = %q / %v", address, err)
	}

	t.Setenv("STEAM_PROFIT_SERVER_ADDR", "127.0.0.1:9090")
	address, err = serverAddressFromEnvironment()
	if err != nil || address != "127.0.0.1:9090" {
		t.Fatalf("explicit address = %q / %v", address, err)
	}
}

func TestServerAddressFromEnvironmentRejectsInvalidPlatformPort(t *testing.T) {
	t.Setenv("STEAM_PROFIT_SERVER_ADDR", "")
	for _, value := range []string{"invalid", "0", "65536"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("PORT", value)
			if _, err := serverAddressFromEnvironment(); err == nil {
				t.Fatalf("PORT %q error = nil", value)
			}
		})
	}
}
