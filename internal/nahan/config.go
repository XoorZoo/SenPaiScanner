package nahan

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all settings for Nahan mode.
type Config struct {
	Mode      string `yaml:"mode"`
	Countries []string `yaml:"countries"`

	Scan struct {
		Workers      int           `yaml:"workers"`
		Timeout      time.Duration `yaml:"timeout"`
		MaxProbes    int           `yaml:"max_probes"`
		MaxResults   int           `yaml:"max_results"`
		Ports        []int         `yaml:"ports"`
		DedupMode    string        `yaml:"dedup_mode"`
		GeoIPEnabled bool          `yaml:"geoip_enabled"`
	} `yaml:"scan"`

	Output struct {
		Directory string `yaml:"directory"`
		TXT       bool   `yaml:"txt"`
		CSV       bool   `yaml:"csv"`
	} `yaml:"output"`

	// Input file (optional)
	InputFile string `yaml:"input_file"`

	// Resolved at runtime
	profiles map[string]CountryProfile
}

// DefaultConfig returns the default Nahan configuration.
func DefaultConfig() *Config {
	c := &Config{}
	c.Mode = "nahan"
	c.Countries = []string{"EG", "NG"}
	c.Scan.Workers = 100
	c.Scan.Timeout = 3 * time.Second
	c.Scan.MaxProbes = 5000
	c.Scan.MaxResults = 100
	c.Scan.Ports = []int{443, 80, 8443, 2053, 2083, 2087, 2096}
	c.Scan.DedupMode = "best-port-only"
	c.Scan.GeoIPEnabled = false
	c.Output.Directory = "./nahan"
	c.Output.TXT = true
	c.Output.CSV = true
	return c
}

// LoadConfig loads configuration from a YAML file.
func LoadConfig(path string) (*Config, error) {
	c := DefaultConfig()
	
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	
	if err := yaml.Unmarshal(data, c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	
	// Apply defaults for zero values
	c.ApplyDefaults()
	return c, nil
}

// ApplyDefaults fills in zero values with defaults.
func (c *Config) ApplyDefaults() {
	if c.Mode == "" {
		c.Mode = "nahan"
	}
	if len(c.Countries) == 0 {
		c.Countries = []string{"EG", "NG"}
	}
	if c.Scan.Workers <= 0 {
		c.Scan.Workers = 100
	}
	if c.Scan.Timeout <= 0 {
		c.Scan.Timeout = 3 * time.Second
	}
	if c.Scan.MaxProbes <= 0 {
		c.Scan.MaxProbes = 5000
	}
	if c.Scan.MaxResults <= 0 {
		c.Scan.MaxResults = 100
	}
	if len(c.Scan.Ports) == 0 {
		c.Scan.Ports = []int{443, 80, 8443, 2053, 2083, 2087, 2096}
	}
	if c.Scan.DedupMode == "" {
		c.Scan.DedupMode = "best-port-only"
	}
	if c.Output.Directory == "" {
		c.Output.Directory = "./nahan"
	}
	if !c.Output.TXT && !c.Output.CSV {
		c.Output.TXT = true
		c.Output.CSV = true
	}
}

// Profiles returns the country profiles map.
func (c *Config) Profiles() map[string]CountryProfile {
	if c.profiles != nil {
		return c.profiles
	}
	c.profiles = make(map[string]CountryProfile)
	for _, code := range c.Countries {
		if p := GetProfile(strings.ToUpper(code), DefaultProfiles()); p != nil {
			c.profiles[code] = *p
		}
	}
	return c.profiles
}

// Flags defines command-line flags for Nahan mode.
func (c *Config) Flags(fs *flag.FlagSet) {
	fs.StringVar(&c.InputFile, "input", "", "Input file (IPs, CIDRs, ranges); empty = embedded CF ranges")
	fs.StringVar(&c.Output.Directory, "output-dir", c.Output.Directory, "Output directory for Nahan files")
	fs.StringVar(&c.Scan.DedupMode, "dedup-mode", c.Scan.DedupMode, "Deduplication mode: best-port-only | all-healthy-ports")
	fs.IntVar(&c.Scan.Workers, "workers", c.Scan.Workers, "Concurrent workers")
	fs.IntVar(&c.Scan.MaxProbes, "max-probes", c.Scan.MaxProbes, "Maximum IPs to probe")
	fs.IntVar(&c.Scan.MaxResults, "max-results", c.Scan.MaxResults, "Maximum healthy results per country")
	fs.DurationVar(&c.Scan.Timeout, "timeout", c.Scan.Timeout, "Probe timeout")
	
	// Custom flag parsers for slices
	fs.Var(portsFlag{&c.Scan.Ports}, "ports", "Ports to test (comma-separated)")
	fs.Var(countriesFlag{&c.Countries}, "country", "Comma-separated country codes (EG,NG,ALL)")
}

// portsFlag implements flag.Value for []int
type portsFlag struct{ v *[]int }

func (p portsFlag) String() string {
	if p.v == nil || len(*p.v) == 0 {
		return ""
	}
	var parts []string
	for _, port := range *p.v {
		parts = append(parts, fmt.Sprint(port))
	}
	return strings.Join(parts, ",")
}

func (p portsFlag) Set(s string) error {
	var ports []int
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		port, err := strconv.Atoi(part)
		if err != nil {
			return fmt.Errorf("invalid port %q: %w", part, err)
		}
		ports = append(ports, port)
	}
	*p.v = ports
	return nil
}

// countriesFlag implements flag.Value for []string
type countriesFlag struct{ v *[]string }

func (c countriesFlag) String() string {
	if c.v == nil || len(*c.v) == 0 {
		return ""
	}
	return strings.Join(*c.v, ",")
}

func (c countriesFlag) Set(s string) error {
	*c.v = ParseCountries(s)
	return nil
}

// ParseCountries parses the --country flag value.
func ParseCountries(s string) []string {
	if strings.ToUpper(s) == "ALL" {
		return AllCodes(DefaultProfiles())
	}
	var result []string
	for _, c := range strings.Split(s, ",") {
		c = strings.TrimSpace(strings.ToUpper(c))
		if c != "" {
			result = append(result, c)
		}
	}
	return result
}

// ExportConfig converts to nahan.ExportConfig.
func (c *Config) ExportConfig() ExportConfig {
	return ExportConfig{
		Directory: c.Output.Directory,
		WriteTXT:  c.Output.TXT,
		WriteCSV:  c.Output.CSV,
		DedupMode: c.Scan.DedupMode,
	}
}