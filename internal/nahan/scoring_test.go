package nahan

import (
	"net"
	"testing"
	"time"

	"github.com/matinsenpai/senpaiscanner/internal/result"
)

func TestCalculateHealthScore_HealthyResult(t *testing.T) {
	r := &result.Result{
		IP:         net.ParseIP("104.16.0.1"),
		Port:       443,
		ProbeMode:  "http",
		Latencies:  []time.Duration{50 * time.Millisecond, 55 * time.Millisecond},
		TLSOk:      true,
		WSOk:       true,
		HTTPStatus: 200,
		Colo:       "CAI",
		Throughput: 12.5 * 1024 * 1024, // 12.5 MB/s = 100 Mbps
		SpeedTested: true,
		Timestamp:  time.Now(),
	}

	profile := &CountryProfile{
		Code:           "EG",
		PreferredColos: []string{"CAI", "ALEX"},
	}
	weights := DefaultWeights()

	score, comps := CalculateHealthScore(r, profile, weights)

	// Should have high score
	if score < 0.7 {
		t.Errorf("Expected score >= 0.7, got %.4f", score)
	}

	// Check components
	if comps.Reachability != 1.0 {
		t.Errorf("Expected reachability 1.0, got %.4f", comps.Reachability)
	}
	if comps.TLS != 1.0 {
		t.Errorf("Expected TLS 1.0, got %.4f", comps.TLS)
	}
	if comps.WS != 1.0 {
		t.Errorf("Expected WS 1.0, got %.4f", comps.WS)
	}
	if comps.ColoBonus != 1.0 {
		t.Errorf("Expected colo bonus 1.0 for CAI, got %.4f", comps.ColoBonus)
	}
}

func TestCalculateHealthScore_UnhealthyResult(t *testing.T) {
	r := &result.Result{
		IP:         net.ParseIP("104.16.0.1"),
		Port:       443,
		ProbeMode:  "http",
		Latencies:  []time.Duration{0, 0},
		TLSOk:      false,
		WSOk:       false,
		HTTPStatus: 0,
		Colo:       "",
		Throughput: 0,
		SpeedTested: false,
		Timestamp:  time.Now(),
	}

	profile := &CountryProfile{
		Code:           "EG",
		PreferredColos: []string{"CAI", "ALEX"},
	}
	weights := DefaultWeights()

	score, comps := CalculateHealthScore(r, profile, weights)

	// Should have low score
	if score > 0.3 {
		t.Errorf("Expected score <= 0.3, got %.4f", score)
	}

	if comps.Reachability != 0.0 {
		t.Errorf("Expected reachability 0.0, got %.4f", comps.Reachability)
	}
}

func TestCalculateHealthScore_ColoBonus(t *testing.T) {
	r := &result.Result{
		IP:         net.ParseIP("104.16.0.1"),
		Port:       443,
		ProbeMode:  "http",
		Latencies:  []time.Duration{100 * time.Millisecond, 110 * time.Millisecond},
		TLSOk:      true,
		WSOk:       false,
		HTTPStatus: 200,
		Colo:       "CAI",
		Throughput: 5 * 1024 * 1024,
		SpeedTested: true,
		Timestamp:  time.Now(),
	}

	// Test with matching colo
	profile1 := &CountryProfile{
		Code:           "EG",
		PreferredColos: []string{"CAI", "ALEX"},
	}
	weights := DefaultWeights()

	score1, comps1 := CalculateHealthScore(r, profile1, weights)
	if comps1.ColoBonus != 1.0 {
		t.Errorf("Expected colo bonus 1.0 for CAI in EG profile, got %.4f", comps1.ColoBonus)
	}

	// Test with non-matching colo
	profile2 := &CountryProfile{
		Code:           "NG",
		PreferredColos: []string{"LOS", "ABV"},
	}

	score2, comps2 := CalculateHealthScore(r, profile2, weights)
	if comps2.ColoBonus != 0.0 {
		t.Errorf("Expected colo bonus 0.0 for CAI in NG profile, got %.4f", comps2.ColoBonus)
	}

	// Score should be higher with colo bonus
	if score1 <= score2 {
		t.Errorf("Expected score with colo bonus (%.4f) > score without (%.4f)", score1, score2)
	}
}

func TestRankByScore(t *testing.T) {
	endpoints := []*Endpoint{
		{IP: "1.2.3.4", Score: 0.5, Country: "EG"},
		{IP: "5.6.7.8", Score: 0.9, Country: "EG"},
		{IP: "9.10.11.12", Score: 0.7, Country: "EG"},
	}

	RankByScore(endpoints)

	if endpoints[0].IP != "5.6.7.8" || endpoints[0].Score != 0.9 {
		t.Errorf("First should be 5.6.7.8 with score 0.9, got %s %.4f", endpoints[0].IP, endpoints[0].Score)
	}
	if endpoints[1].IP != "9.10.11.12" || endpoints[1].Score != 0.7 {
		t.Errorf("Second should be 9.10.11.12 with score 0.7, got %s %.4f", endpoints[1].IP, endpoints[1].Score)
	}
	if endpoints[2].IP != "1.2.3.4" || endpoints[2].Score != 0.5 {
		t.Errorf("Third should be 1.2.3.4 with score 0.5, got %s %.4f", endpoints[2].IP, endpoints[2].Score)
	}
}