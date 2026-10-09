package nahan

import (
	"context"
	"fmt"
	"net"
)

// NoopGeoIPClient is a no-op implementation for when GeoIP is disabled.
type NoopGeoIPClient struct{}

func (n *NoopGeoIPClient) Country(ip net.IP) (string, float64, error) {
	return "", 0, fmt.Errorf("geoip disabled")
}

// NoopASNClient is a no-op implementation for when ASN lookup is disabled.
type NoopASNClient struct{}

func (n *NoopASNClient) ASN(ip net.IP) (int, string, error) {
	return 0, "", fmt.Errorf("asn lookup disabled")
}

// NoopGeoIPClientCtx implements GeoIPClientWithContext.
type NoopGeoIPClientCtx struct{}

func (n *NoopGeoIPClientCtx) Country(ctx context.Context, ip net.IP) (string, float64, error) {
	return "", 0, fmt.Errorf("geoip disabled")
}

// NoopASNClientCtx implements ASNClientWithContext.
type NoopASNClientCtx struct{}

func (n *NoopASNClientCtx) ASN(ctx context.Context, ip net.IP) (int, string, error) {
	return 0, "", fmt.Errorf("asn lookup disabled")
}