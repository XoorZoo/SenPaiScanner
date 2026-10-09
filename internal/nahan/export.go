package nahan

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ExportConfig controls export behavior.
type ExportConfig struct {
	Directory   string // output directory
	WriteTXT    bool   // write nahan-<country>.txt files
	WriteCSV    bool   // write nahan-results.csv
	DedupMode   string // "best-port-only" or "all-healthy-ports"
}

// DefaultExportConfig returns default export settings.
func DefaultExportConfig() ExportConfig {
	return ExportConfig{
		Directory: "./nahan",
		WriteTXT:  true,
		WriteCSV:  true,
		DedupMode: "best-port-only",
	}
}

// WriteExports writes Nahan export files.
func WriteExports(endpoints []*Endpoint, cfg ExportConfig) error {
	if err := os.MkdirAll(cfg.Directory, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	// Deduplicate
	deduped := Deduplicate(endpoints, cfg.DedupMode)

	// Group by country
	byCountry := make(map[string][]*Endpoint)
	for _, ep := range deduped {
		byCountry[ep.Country] = append(byCountry[ep.Country], ep)
	}

	// Assign deterministic indices per country
	for _, eps := range byCountry {
		sort.Slice(eps, func(i, j int) bool {
			return eps[i].Score > eps[j].Score
		})
		for i, ep := range eps {
			ep.Index = i + 1
		}
	}

	// Write TXT files per country
	if cfg.WriteTXT {
		for country, eps := range byCountry {
			if country == "UNKNOWN" {
				continue // skip UNKNOWN in TXT
			}
			txtPath := filepath.Join(cfg.Directory, fmt.Sprintf("nahan-%s.txt", strings.ToLower(country)))
			if err := writeTXT(txtPath, eps, country); err != nil {
				return err
			}
		}
	}

	// Write CSV with all endpoints
	if cfg.WriteCSV {
		csvPath := filepath.Join(cfg.Directory, "nahan-results.csv")
		if err := writeCSV(csvPath, deduped); err != nil {
			return err
		}
	}

	return nil
}

// writeTXT writes the simple IP#CC-NN format.
func writeTXT(path string, endpoints []*Endpoint, country string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()

	for _, ep := range endpoints {
		line := fmt.Sprintf("%s#%s-%02d\n", ep.IP, strings.ToUpper(country), ep.Index)
		if _, err := f.WriteString(line); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
	}
	return nil
}

// writeCSV writes the detailed CSV export.
func writeCSV(path string, endpoints []*Endpoint) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Header
	header := []string{
		"country", "ip", "port", "colo", "asn", "isp",
		"latency_ms", "loss_pct", "throughput_mbps",
		"score", "status", "probe_mode",
		"tls_success", "ws_success", "country_confidence",
	}
	if err := w.Write(header); err != nil {
		return err
	}

	// Sort: by country, then by score desc
	sorted := make([]*Endpoint, len(endpoints))
	copy(sorted, endpoints)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Country != sorted[j].Country {
			return sorted[i].Country < sorted[j].Country
		}
		return sorted[i].Score > sorted[j].Score
	})

	for _, ep := range sorted {
		row := []string{
			ep.Country,
			ep.IP,
			fmt.Sprintf("%d", ep.Port),
			ep.Colo,
			fmt.Sprintf("%d", ep.ASN),
			ep.ISP,
			fmt.Sprintf("%.2f", float64(ep.Latency.Milliseconds())),
			fmt.Sprintf("%.1f", ep.Loss),
			fmt.Sprintf("%.2f", ep.Throughput*8/1_000_000),
			fmt.Sprintf("%.4f", ep.Score),
			healthStatus(ep),
			ep.ProbeMode,
			fmt.Sprintf("%v", ep.TLSSuccess),
			fmt.Sprintf("%v", ep.WSSuccess),
			fmt.Sprintf("%.2f", ep.CountryConfidence),
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return nil
}

// healthStatus returns a human-readable status.
func healthStatus(ep *Endpoint) string {
	if ep.Score >= 0.7 {
		return "healthy"
	} else if ep.Score >= 0.4 {
		return "degraded"
	}
	return "unhealthy"
}

// Deduplicate removes duplicate IPs based on the configured mode.
func Deduplicate(endpoints []*Endpoint, mode string) []*Endpoint {
	if mode == "all-healthy-ports" {
		return endpoints // keep all
	}

	// best-port-only: keep highest score per IP
	best := make(map[string]*Endpoint)
	for _, ep := range endpoints {
		key := ep.IP
		if existing, ok := best[key]; !ok || ep.Score > existing.Score {
			best[key] = ep
		}
	}

	result := make([]*Endpoint, 0, len(best))
	for _, ep := range best {
		result = append(result, ep)
	}
	return result
}