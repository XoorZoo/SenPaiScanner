package nahan

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteExports(t *testing.T) {
	tmpDir := t.TempDir()
	
	endpoints := []*Endpoint{
		{IP: "104.16.0.1", Port: 443, Country: "EG", Score: 0.9, Colo: "CAI", ASN: 13335, ISP: "Cloudflare", Latency: 50, Loss: 0, Throughput: 10000000, ProbeMode: "http", TLSSuccess: true, WSSuccess: true, CountryConfidence: 0.9},
		{IP: "104.16.0.2", Port: 443, Country: "EG", Score: 0.8, Colo: "CAI", ASN: 13335, ISP: "Cloudflare", Latency: 60, Loss: 0, Throughput: 8000000, ProbeMode: "http", TLSSuccess: true, WSSuccess: true, CountryConfidence: 0.8},
		{IP: "104.16.0.3", Port: 443, Country: "NG", Score: 0.85, Colo: "LOS", ASN: 13335, ISP: "Cloudflare", Latency: 55, Loss: 0, Throughput: 9000000, ProbeMode: "http", TLSSuccess: true, WSSuccess: true, CountryConfidence: 0.85},
		{IP: "104.16.0.1", Port: 80, Country: "EG", Score: 0.7, Colo: "CAI", ASN: 13335, ISP: "Cloudflare", Latency: 40, Loss: 0, Throughput: 5000000, ProbeMode: "http", TLSSuccess: false, WSSuccess: false, CountryConfidence: 0.7},
	}

	cfg := ExportConfig{
		Directory: tmpDir,
		WriteTXT:  true,
		WriteCSV:  true,
		DedupMode: "best-port-only",
	}

	err := WriteExports(endpoints, cfg)
	if err != nil {
		t.Fatalf("WriteExports failed: %v", err)
	}

	// Check EG txt
	egTxt := filepath.Join(tmpDir, "nahan-eg.txt")
	content, err := os.ReadFile(egTxt)
	if err != nil {
		t.Fatalf("Failed to read EG txt: %v", err)
	}
	
	expectedLines := []string{
		"104.16.0.1#EG-01",
		"104.16.0.2#EG-02",
	}
	for _, line := range expectedLines {
		if !contains(string(content), line) {
			t.Errorf("Expected line %q not found in EG txt", line)
		}
	}

	// Check NG txt
	ngTxt := filepath.Join(tmpDir, "nahan-ng.txt")
	content, err = os.ReadFile(ngTxt)
	if err != nil {
		t.Fatalf("Failed to read NG txt: %v", err)
	}
	if !contains(string(content), "104.16.0.3#NG-01") {
		t.Errorf("Expected NG endpoint not found")
	}

	// Check CSV
	csvPath := filepath.Join(tmpDir, "nahan-results.csv")
	content, err = os.ReadFile(csvPath)
	if err != nil {
		t.Fatalf("Failed to read CSV: %v", err)
	}
	csvContent := string(content)
	if !contains(csvContent, "country,ip,port") {
		t.Errorf("CSV header missing")
	}
	if !contains(csvContent, "104.16.0.1") {
		t.Errorf("CSV missing endpoint")
	}
}

func TestDeduplicate_BestPortOnly(t *testing.T) {
	endpoints := []*Endpoint{
		{IP: "1.2.3.4", Port: 443, Score: 0.9},
		{IP: "1.2.3.4", Port: 80, Score: 0.7},
		{IP: "5.6.7.8", Port: 443, Score: 0.8},
	}

	result := Deduplicate(endpoints, "best-port-only")
	
	if len(result) != 2 {
		t.Errorf("Expected 2 endpoints after dedup, got %d", len(result))
	}
	
	if result[0].IP == "1.2.3.4" && result[0].Score != 0.9 {
		t.Errorf("Expected best port (443) with score 0.9, got %.2f", result[0].Score)
	}
}

func TestDeduplicate_AllHealthyPorts(t *testing.T) {
	endpoints := []*Endpoint{
		{IP: "1.2.3.4", Port: 443, Score: 0.9},
		{IP: "1.2.3.4", Port: 80, Score: 0.7},
		{IP: "5.6.7.8", Port: 443, Score: 0.8},
	}

	result := Deduplicate(endpoints, "all-healthy-ports")
	
	if len(result) != 3 {
		t.Errorf("Expected 3 endpoints (all kept), got %d", len(result))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}