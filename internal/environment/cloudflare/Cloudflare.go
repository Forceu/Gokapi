package cloudflare

import (
	"bufio"
	"context"
	_ "embed"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	cloudflareIPv4URL      = "https://www.cloudflare.com/ips-v4"
	cloudflareIPv6URL      = "https://www.cloudflare.com/ips-v6"
	cloudflareFetchTimeout = 10 * time.Second
	maxCloudflareListSize  = 64 * 1024 // the real lists are well below 1 KB
)

// Fallback lists, compiled into the binary. Used if the download fails.
//
//go:embed ips-v4.txt
var embeddedIPv4 string

//go:embed ips-v6.txt
var embeddedIPv6 string

// GetIpRanges returns the IPv4 and IPv6 ranges of Cloudflare as non-empty
// lines in CIDR notation, e.g. "173.245.48.0/20" (IPv4 first, then IPv6).
//
// Each list is downloaded from Cloudflare. If a download fails for any reason
// (network error, bad status, invalid content), the embedded copy of that list
// is used instead. An error is only returned if the embedded copy is invalid too.
func GetIpRanges() ([]string, error) {
	fmt.Println("Downloading Cloudflare IP ranges...")
	parsedIps, err := fetchCloudflareIPs(context.Background())
	return parsedIps, err
}

// fetchCloudflareIPs returns the IPv4 and IPv6 ranges of Cloudflare as non-empty
// lines in CIDR notation, e.g. "173.245.48.0/20" (IPv4 first, then IPv6).
//
// Each list is downloaded from Cloudflare. If a download fails for any reason
// (network error, bad status, invalid content), the embedded copy of that list
// is used instead. An error is only returned if the embedded copy is invalid too.
func fetchCloudflareIPs(ctx context.Context) ([]string, error) {
	client := &http.Client{Timeout: cloudflareFetchTimeout}

	lists := []struct {
		url      string
		embedded string
		name     string
	}{
		{cloudflareIPv4URL, embeddedIPv4, "ips-v4.txt"},
		{cloudflareIPv6URL, embeddedIPv6, "ips-v6.txt"},
	}

	var result []string
	for _, l := range lists {
		ranges, err := downloadIpList(ctx, client, l.url)
		if err != nil {
			fmt.Printf("WARNING: Unable to download Cloudflare %s. Using embedded IP rage. %s\n", l.name, err)
			ranges, err = parseIpList(strings.NewReader(l.embedded), "embedded "+l.name)
			if err != nil {
				fmt.Printf("WARNING: Unable to parse embedded Cloudflare %s. %s\n", l.name, err)
				return nil, err
			}
		}
		result = append(result, ranges...)
	}
	return result, nil
}

// downloadIpList downloads a single list and returns its non-empty, validated lines
func downloadIpList(ctx context.Context, client *http.Client, url string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for %s: %w", url, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading %s: unexpected status %s", url, resp.Status)
	}
	return parseIpList(io.LimitReader(resp.Body, maxCloudflareListSize), url)
}

// parseIpList reads a list with one CIDR range per line and returns the non-empty lines.
// source is only used in error messages.
func parseIpList(r io.Reader, source string) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// These lines end up in a list of trusted proxies, so reject anything
		// that is not a valid CIDR range instead of passing it on
		if _, _, err := net.ParseCIDR(line); err != nil {
			return nil, fmt.Errorf("%s contains an invalid range %q: %w", source, line, err)
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading %s: %w", source, err)
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("%s contains no IP ranges", source)
	}
	return lines, nil
}
