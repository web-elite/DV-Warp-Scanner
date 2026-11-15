package main

import "testing"

func TestXrayVersionTagValidation(t *testing.T) {
	valid := []string{"v26.11.9", "v1.2.3-beta.1", "v2026.11"}
	for _, tag := range valid {
		if !versionPattern.MatchString(tag) {
			t.Errorf("expected %q to be accepted", tag)
		}
	}
	invalid := []string{"", "26.11.9", "../v1", "v1/../../xray", "latest"}
	for _, tag := range invalid {
		if versionPattern.MatchString(tag) {
			t.Errorf("expected %q to be rejected", tag)
		}
	}
}

func TestEqualBytes(t *testing.T) {
	if !equalBytes([]byte{1, 2, 3}, []byte{1, 2, 3}) {
		t.Fatal("equal byte slices should match")
	}
	if equalBytes([]byte{1, 2, 3}, []byte{1, 2, 4}) {
		t.Fatal("different byte slices must not match")
	}
	if equalBytes([]byte{1}, []byte{1, 2}) {
		t.Fatal("byte slices with different lengths must not match")
	}
}

func TestBuildConfigUsesCachedWarpParameters(t *testing.T) {
	previousConfig := scanConfig
	previousWarp := activeWarpParams
	t.Cleanup(func() {
		scanConfig = previousConfig
		activeWarpParams = previousWarp
	})

	scanConfig = ScanConfig{
		Ipv4Mode: true,
		UseNoise: false,
		Endpoints: []string{"162.159.192.1:2408"},
	}
	activeWarpParams = &WarpParams{
		IPv6:       "2606:4700:110:8c2a::1/128",
		Reserved:   []int{1, 2, 3},
		PublicKey:  "test-public-key",
		PrivateKey: "test-private-key",
	}

	config, err := buildConfig()
	if err != nil {
		t.Fatalf("buildConfig returned an error: %v", err)
	}
	if len(config.Inbounds) != 1 {
		t.Fatalf("expected one inbound, got %d", len(config.Inbounds))
	}
	if config.Inbounds[0].Port != 1080 {
		t.Fatalf("expected first inbound port 1080, got %d", config.Inbounds[0].Port)
	}
	if len(config.Outbounds) != 2 {
		t.Fatalf("expected direct and one WireGuard outbound, got %d", len(config.Outbounds))
	}
	wireguard, ok := config.Outbounds[1].(WgOutbound)
	if !ok {
		t.Fatalf("expected WireGuard outbound, got %T", config.Outbounds[1])
	}
	if wireguard.Settings.SecretKey != "test-private-key" {
		t.Fatal("cached WARP private key was not applied")
	}
}

func TestExtractWarpParamsRejectsMissingPeer(t *testing.T) {
	var config WarpConfig
	config.Config.Interface.Addresses.V6 = "2606:4700:110:8c2a::1"
	config.Config.ClientID = "AQID"

	if _, err := extractWarpParams(config, "private-key"); err == nil {
		t.Fatal("expected an error when the WARP response has no peer")
	}
}
