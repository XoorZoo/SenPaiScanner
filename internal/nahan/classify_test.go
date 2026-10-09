package nahan

import (
	"context"
	"net"
	"testing"

	res "github.com/matinsenpai/senpaiscanner/internal/result"
)

func TestClassifier_ColoBased(t *testing.T) {
	profiles := DefaultProfiles()
	classifier := &Classifier{
		Profiles: profiles,
	}

	// Test EG colo
	r := &res.Result{IP: net.ParseIP("104.16.0.1"), Colo: "CAI"}
	result := classifier.ClassifyWithContext(context.Background(), r)
	if result.Country != "EG" {
		t.Errorf("Expected EG for CAI colo, got %s", result.Country)
	}
	if result.Method != "colo" {
		t.Errorf("Expected method colo, got %s", result.Method)
	}
	if result.Confidence != 0.7 {
		t.Errorf("Expected confidence 0.7, got %.2f", result.Confidence)
	}

	// Test NG colo
	r = &res.Result{IP: net.ParseIP("104.16.0.2"), Colo: "LOS"}
	result = classifier.ClassifyWithContext(context.Background(), r)
	if result.Country != "NG" {
		t.Errorf("Expected NG for LOS colo, got %s", result.Country)
	}

	// Test unknown colo
	r = &res.Result{IP: net.ParseIP("104.16.0.3"), Colo: "XYZ"}
	result = classifier.ClassifyWithContext(context.Background(), r)
	if result.Country != "UNKNOWN" {
		t.Errorf("Expected UNKNOWN for XYZ colo, got %s", result.Country)
	}
}

func TestClassifier_ASNBased(t *testing.T) {
	profiles := DefaultProfiles()
	
	// Mock ASN client that returns Egyptian ISP
	mockASN := &mockASNClient{
		asn: 12345,
		org: "Telecom Egypt",
	}
	
	classifier := &Classifier{
		Profiles:     profiles,
		ASNClientCtx: mockASN,
	}

	r := &res.Result{IP: net.ParseIP("104.16.0.1")}
	result := classifier.ClassifyWithContext(context.Background(), r)
	
	if result.Country != "EG" {
		t.Errorf("Expected EG for Telecom Egypt, got %s", result.Country)
	}
	if result.Method != "asn" {
		t.Errorf("Expected method asn, got %s", result.Method)
	}
	if result.ASN != 12345 {
		t.Errorf("Expected ASN 12345, got %d", result.ASN)
	}
	if result.ISP != "Telecom Egypt" {
		t.Errorf("Expected ISP Telecom Egypt, got %s", result.ISP)
	}

	// Test NG ISP
	mockASN = &mockASNClient{
		asn: 54321,
		org: "MTN Nigeria",
	}
	classifier = &Classifier{
		Profiles:     profiles,
		ASNClientCtx: mockASN,
	}
	r = &res.Result{IP: net.ParseIP("104.16.0.2")}
	result = classifier.ClassifyWithContext(context.Background(), r)
	if result.Country != "NG" {
		t.Errorf("Expected NG for MTN Nigeria, got %s", result.Country)
	}
}

func TestClassifier_Unknown(t *testing.T) {
	profiles := DefaultProfiles()
	classifier := &Classifier{
		Profiles:     profiles,
		ASNClientCtx: &mockASNClient{asn: 99999, org: "Unknown ISP"},
	}

	r := &res.Result{IP: net.ParseIP("104.16.0.1")}
	result := classifier.ClassifyWithContext(context.Background(), r)
	
	if result.Country != "UNKNOWN" {
		t.Errorf("Expected UNKNOWN for unknown ISP, got %s", result.Country)
	}
	if result.Method != "asn" {
		t.Errorf("Expected method asn, got %s", result.Method)
	}
}

type mockASNClient struct {
	asn int
	org string
}

func (m *mockASNClient) ASN(ctx context.Context, ip net.IP) (int, string, error) {
	return m.asn, m.org, nil
}