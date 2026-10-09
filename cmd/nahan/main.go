package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/matinsenpai/senpaiscanner/internal/antidpi"
	"github.com/matinsenpai/senpaiscanner/internal/engine"
	"github.com/matinsenpai/senpaiscanner/internal/ipsrc"
	"github.com/matinsenpai/senpaiscanner/internal/nahan"
	"github.com/matinsenpai/senpaiscanner/internal/prober"
	"github.com/matinsenpai/senpaiscanner/internal/result"
	"github.com/matinsenpai/senpaiscanner/internal/scanjob"
	"github.com/matinsenpai/senpaiscanner/pkg/version"
)

func main() {
	cfg := nahan.DefaultConfig()
	cfg.Flags(flag.CommandLine)
	
	configFile := flag.String("config", "", "Path to YAML config file")
	flag.Parse()

	// Load config file if provided
	if *configFile != "" {
		loaded, err := nahan.LoadConfig(*configFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}
		cfg = loaded
	}

	// Parse countries from flag
	cfg.Countries = nahan.ParseCountries(strings.Join(cfg.Countries, ","))
	cfg.ApplyDefaults()

	fmt.Println("SenPai Scanner - Nahan Mode")
	fmt.Println("Version:", version.String())
	fmt.Println("Target countries:", strings.Join(cfg.Countries, ", "))
	fmt.Println("Max probes:", cfg.Scan.MaxProbes)
	fmt.Println("Max results per country:", cfg.Scan.MaxResults)
	fmt.Println("Workers:", cfg.Scan.Workers)
	fmt.Println("Timeout:", cfg.Scan.Timeout)
	fmt.Println("Output dir:", cfg.Output.Directory)
	fmt.Println()

	// Build IP source
	src, err := buildIPSource(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build IP source: %v\n", err)
		os.Exit(1)
	}

	// Create classifier (colo primary, ASN fallback optional)
	classifier := &nahan.Classifier{
		Profiles: cfg.Profiles(),
	}

	// Run scan
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	results, err := runNahanScan(ctx, cfg, src, classifier)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Scan failed: %v\n", err)
		os.Exit(1)
	}

	// Export results
	exportCfg := cfg.ExportConfig()
	if err := nahan.WriteExports(results, exportCfg); err != nil {
		fmt.Fprintf(os.Stderr, "Export failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nDone!")
	fmt.Println("Output files in:", cfg.Output.Directory)
	for _, country := range cfg.Countries {
		if cfg.Output.TXT {
			fmt.Printf("  - nahan-%s.txt\n", strings.ToLower(country))
		}
	}
	if cfg.Output.CSV {
		fmt.Println("  - nahan-results.csv")
	}
}

func buildIPSource(cfg *nahan.Config) (*ipsrc.Source, error) {
	var extra []string
	
	if cfg.InputFile != "" {
		// Load IPs/CIDRs from file
		ips, cidrs, err := scanjob.LoadTargetsFromFile(cfg.InputFile)
		if err != nil {
			return nil, fmt.Errorf("load targets: %w", err)
		}
		// Convert IPs to /32 CIDRs
		for _, ip := range ips {
			extra = append(extra, ip.String()+"/32")
		}
		extra = append(extra, cidrs...)
		fmt.Printf("Loaded %d IPs and %d CIDRs from %s\n", len(ips), len(cidrs), cfg.InputFile)
	}

	// Use embedded Cloudflare ranges if no explicit input
	useBuiltin := cfg.InputFile == ""
	return ipsrc.NewWithOptions(true, false, extra, ipsrc.Options{UseBuiltin: useBuiltin})
}

func runNahanScan(ctx context.Context, cfg *nahan.Config, src *ipsrc.Source, classifier *nahan.Classifier) ([]*nahan.Endpoint, error) {
	// Probe configuration
	probeCfg := prober.Config{
		Mode:               prober.ModeHTTP,
		Tries:              2, // Fast initial probe
		Timeout:            cfg.Scan.Timeout,
		SNI:                "speed.cloudflare.com",
		SpeedBytes:         0, // No download test in phase 1
		InsecureSkipVerify: true,
		RequireWebSocket:   false,
		AntiDPI:            antidpi.DefaultProfile(),
	}

	// Engine configuration
	engCfg := engine.Config{
		Concurrency: cfg.Scan.Workers,
		ProbeConfig: probeCfg,
	}
	eng := engine.New(engCfg)

	// Stream IPs
	ipStream := src.MahsaNGV4Stream(ctx, cfg.Scan.MaxProbes)

	// Results collection
	var (
		mu           sync.Mutex
		allResults   []*result.Result
		countryHealthy = make(map[string]int64)
		tested       atomic.Int64
		probedCount  int
	)

	// Initialize country counters
	for _, c := range cfg.Countries {
		countryHealthy[strings.ToUpper(c)] = 0
	}

	// Progress display
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-ticker.C:
				fmt.Printf("\rProbed: %d  Healthy: ", tested.Load())
				for _, c := range cfg.Countries {
					fmt.Printf("%s=%d ", c, countryHealthy[strings.ToUpper(c)])
				}
			case <-done:
				fmt.Printf("\rProbed: %d  Healthy: ", tested.Load())
				for _, c := range cfg.Countries {
					fmt.Printf("%s=%d ", c, countryHealthy[strings.ToUpper(c)])
				}
				fmt.Println()
				return
			}
		}
	}()

	// Run engine
	eng.Run(ctx, ipStream, func(r *result.Result) {
		tested.Add(1)
		probedCount++

		if !r.IsHealthy() {
			return
		}

		// Classify country
		classification := classifier.ClassifyWithContext(ctx, r)
		r.Country = classification.Country
		r.CountryConfidence = classification.Confidence
		r.ASN = classification.ASN
		r.ISP = classification.ISP

		// Calculate health score
		profile := classifier.Profiles[r.Country]
		weights := nahan.DefaultWeights()
		score, _ := nahan.CalculateHealthScore(r, &profile, weights)
		r.HealthScore = score

		// Store TLS/WS success
		r.TLSSuccess = r.TLSOk
		r.WSSuccess = r.WSOk

		mu.Lock()
		allResults = append(allResults, r)
		mu.Unlock()

		// Per-country early exit
		country := strings.ToUpper(r.Country)
		if _, ok := countryHealthy[country]; ok {
			countryHealthy[country]++
			allReached := true
			for _, c := range cfg.Countries {
				if countryHealthy[strings.ToUpper(c)] < int64(cfg.Scan.MaxResults) {
					allReached = false
					break
				}
			}
			if allReached {
				cancel()
			}
		}
	})

	close(done)

	// Convert to Nahan endpoints
	endpoints := make([]*nahan.Endpoint, 0, len(allResults))
	for _, r := range allResults {
		ep := &nahan.Endpoint{
			IP:                r.IP.String(),
			Port:              r.Port,
			Country:           r.Country,
			Score:             r.HealthScore,
			Colo:              r.Colo,
			ASN:               r.ASN,
			ISP:               r.ISP,
			Latency:           r.Avg(),
			Loss:              r.Loss(),
			Throughput:        r.Throughput,
			Components:        nahan.ScoreComponents{},
			ProbeMode:         r.ProbeMode,
			TLSSuccess:        r.TLSSuccess,
			WSSuccess:         r.WSSuccess,
			CountryConfidence: r.CountryConfidence,
		}
		endpoints = append(endpoints, ep)
	}

	// Filter by target countries
	filtered := filterEndpoints(endpoints, cfg)

	fmt.Printf("\nScan complete. Total probed: %d, Healthy: %d, After filtering: %d\n",
		probedCount, len(allResults), len(filtered))

	return filtered, nil
}

func filterEndpoints(endpoints []*nahan.Endpoint, cfg *nahan.Config) []*nahan.Endpoint {
	// Build allowed countries set
	allowed := make(map[string]bool)
	for _, c := range cfg.Countries {
		allowed[strings.ToUpper(c)] = true
	}

	// Filter by country
	var filtered []*nahan.Endpoint
	for _, ep := range endpoints {
		if allowed[ep.Country] || (ep.Country == "UNKNOWN" && allowed["UNKNOWN"]) {
			filtered = append(filtered, ep)
		}
	}

	// Sort by country then score desc
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].Country != filtered[j].Country {
			return filtered[i].Country < filtered[j].Country
		}
		return filtered[i].Score > filtered[j].Score
	})

	return filtered
}