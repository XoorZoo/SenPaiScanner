package nahan

import (
	"math"
	"time"

	"github.com/matinsenpai/senpaiscanner/internal/result"
)

// Weights for health score calculation.
type ScoreWeights struct {
	Reachability float64
	Latency      float64
	Loss         float64
	Throughput   float64
	TLS          float64
	WS           float64
	ColoBonus    float64
}

// DefaultWeights returns the default scoring weights.
func DefaultWeights() ScoreWeights {
	return ScoreWeights{
		Reachability: 0.30,
		Latency:      0.15,
		Loss:         0.15,
		Throughput:   0.15,
		TLS:          0.10,
		WS:           0.10,
		ColoBonus:    0.05,
	}
}

// ScoreComponents holds the individual score components for debugging.
type ScoreComponents struct {
	Reachability float64
	Latency      float64
	Loss         float64
	Throughput   float64
	TLS          float64
	WS           float64
	ColoBonus    float64
	Total        float64
}

// CalculateHealthScore computes a health score for a result.
// Returns the total score (0.0-1.0) and the component breakdown.
func CalculateHealthScore(r *result.Result, profile *CountryProfile, weights ScoreWeights) (float64, ScoreComponents) {
	var comps ScoreComponents

	// 1. Reachability: 1.0 if healthy, 0.0 otherwise
	if r.IsHealthy() {
		comps.Reachability = 1.0
	} else {
		comps.Reachability = 0.0
	}

	// 2. Latency: normalized (lower is better), capped at 500ms
	if r.Avg() > 0 {
		latencyMs := float64(r.Avg().Milliseconds())
		comps.Latency = 1.0 - math.Min(latencyMs/500.0, 1.0)
	} else {
		comps.Latency = 0.0
	}

	// 3. Loss: 1.0 - loss%/100
	comps.Loss = 1.0 - r.Loss()/100.0

	// 4. Throughput: log scale normalization (higher is better)
	// 10 Mbps = ~0.5, 100 Mbps = ~0.7, 1000 Mbps = ~1.0
	if r.Throughput > 0 {
		mbps := r.Throughput * 8 / 1_000_000
		comps.Throughput = math.Min(math.Log10(mbps+1)/math.Log10(1000+1), 1.0)
	} else {
		comps.Throughput = 0.0
	}

	// 5. TLS success
	if r.TLSOk {
		comps.TLS = 1.0
	} else {
		comps.TLS = 0.0
	}

	// 6. WebSocket success
	if r.WSOk {
		comps.WS = 1.0
	} else {
		comps.WS = 0.0
	}

	// 7. Colo bonus: if colo is in preferred list for the country
	comps.ColoBonus = 0.0
	if profile != nil && r.Colo != "" {
		for _, preferred := range profile.PreferredColos {
			if r.Colo == preferred {
				comps.ColoBonus = 1.0
				break
			}
		}
	}

	// Weighted total
	comps.Total = comps.Reachability*weights.Reachability +
		comps.Latency*weights.Latency +
		comps.Loss*weights.Loss +
		comps.Throughput*weights.Throughput +
		comps.TLS*weights.TLS +
		comps.WS*weights.WS +
		comps.ColoBonus*weights.ColoBonus

	return comps.Total, comps
}

// RankByScore sorts endpoints by health score descending.
func RankByScore(endpoints []*Endpoint) {
	// Simple insertion sort for small slices, stable
	for i := 1; i < len(endpoints); i++ {
		j := i
		for j > 0 && endpoints[j].Score > endpoints[j-1].Score {
			endpoints[j], endpoints[j-1] = endpoints[j-1], endpoints[j]
			j--
		}
	}
}

// Endpoint represents a Nahan-ready endpoint with enriched metadata.
type Endpoint struct {
	IP              string
	Port            int
	Country         string
	Index           int     // 1-based deterministic index per country
	Score           float64
	Colo            string
	ASN             int
	ISP             string
	Latency         time.Duration
	Loss            float64
	Throughput      float64 // bytes/sec
	Components      ScoreComponents
	ProbeMode       string
	TLSSuccess      bool
	WSSuccess       bool
	CountryConfidence float64
}