package nahan

import (
	"flag"
	"testing"
)

func TestDefaultProfiles(t *testing.T) {
	profiles := DefaultProfiles()
	
	if len(profiles) != 2 {
		t.Errorf("Expected 2 default profiles, got %d", len(profiles))
	}
	
	eg, ok := profiles["EG"]
	if !ok {
		t.Error("EG profile missing")
	}
	if eg.Name != "Egypt" {
		t.Errorf("Expected Egypt, got %s", eg.Name)
	}
	if len(eg.PreferredColos) == 0 {
		t.Error("EG should have preferred colos")
	}
	
	ng, ok := profiles["NG"]
	if !ok {
		t.Error("NG profile missing")
	}
	if ng.Name != "Nigeria" {
		t.Errorf("Expected Nigeria, got %s", ng.Name)
	}
}

func TestGetProfile(t *testing.T) {
	profiles := DefaultProfiles()
	
	p := GetProfile("EG", profiles)
	if p == nil {
		t.Error("GetProfile(EG) returned nil")
	}
	if p.Code != "EG" {
		t.Errorf("Expected EG, got %s", p.Code)
	}
	
	p = GetProfile("XX", profiles)
	if p != nil {
		t.Error("GetProfile(XX) should return nil")
	}
}

func TestAllCodes(t *testing.T) {
	profiles := DefaultProfiles()
	codes := AllCodes(profiles)
	
	if len(codes) != 2 {
		t.Errorf("Expected 2 codes, got %d", len(codes))
	}
	
	foundEG := false
	foundNG := false
	for _, c := range codes {
		if c == "EG" { foundEG = true }
		if c == "NG" { foundNG = true }
	}
	if !foundEG || !foundNG {
		t.Error("Missing expected codes")
	}
}

func TestParseCountries(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"EG,NG", []string{"EG", "NG"}},
		{"eg,ng", []string{"EG", "NG"}},
		{"EG, NG, EG", []string{"EG", "NG", "EG"}},
		{"ALL", []string{"EG", "NG"}},
		{"all", []string{"EG", "NG"}},
		{"", []string{}},
	}
	
	for _, tt := range tests {
		result := ParseCountries(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("ParseCountries(%q): expected %v, got %v", tt.input, tt.expected, result)
		}
	}
}

func TestConfig_Defaults(t *testing.T) {
	cfg := DefaultConfig()
	
	if cfg.Mode != "nahan" {
		t.Errorf("Expected mode nahan, got %s", cfg.Mode)
	}
	if len(cfg.Countries) != 2 {
		t.Errorf("Expected 2 default countries, got %d", len(cfg.Countries))
	}
	if cfg.Scan.Workers != 100 {
		t.Errorf("Expected 100 workers, got %d", cfg.Scan.Workers)
	}
	if cfg.Scan.MaxProbes != 5000 {
		t.Errorf("Expected 5000 max probes, got %d", cfg.Scan.MaxProbes)
	}
	if cfg.Scan.MaxResults != 100 {
		t.Errorf("Expected 100 max results, got %d", cfg.Scan.MaxResults)
	}
	if cfg.Scan.DedupMode != "best-port-only" {
		t.Errorf("Expected best-port-only, got %s", cfg.Scan.DedupMode)
	}
}

func TestConfig_Flags(t *testing.T) {
	cfg := DefaultConfig()
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cfg.Flags(fs)
	
	// Test default values are set
	if fs.Lookup("input") == nil {
		t.Error("input flag missing")
	}
	if fs.Lookup("output-dir") == nil {
		t.Error("output-dir flag missing")
	}
	if fs.Lookup("dedup-mode") == nil {
		t.Error("dedup-mode flag missing")
	}
	if fs.Lookup("workers") == nil {
		t.Error("workers flag missing")
	}
	if fs.Lookup("max-probes") == nil {
		t.Error("max-probes flag missing")
	}
	if fs.Lookup("max-results") == nil {
		t.Error("max-results flag missing")
	}
	if fs.Lookup("timeout") == nil {
		t.Error("timeout flag missing")
	}
	if fs.Lookup("ports") == nil {
		t.Error("ports flag missing")
	}
	if fs.Lookup("country") == nil {
		t.Error("country flag missing")
	}
}

func TestConfig_ExportConfig(t *testing.T) {
	cfg := DefaultConfig()
	exp := cfg.ExportConfig()
	
	if exp.Directory != "./nahan" {
		t.Errorf("Expected ./nahan, got %s", exp.Directory)
	}
	if !exp.WriteTXT {
		t.Error("Expected WriteTXT=true")
	}
	if !exp.WriteCSV {
		t.Error("Expected WriteCSV=true")
	}
	if exp.DedupMode != "best-port-only" {
		t.Errorf("Expected best-port-only, got %s", exp.DedupMode)
	}
}