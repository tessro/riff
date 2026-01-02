package sonos

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	ssdpAddr      = "239.255.255.250:1900"
	sonosURN      = "urn:schemas-upnp-org:device:ZonePlayer:1"
	defaultTTL    = 5 * time.Minute
	fileCacheTTL  = 5 * time.Minute
)

var mSearchRequest = []byte(
	"M-SEARCH * HTTP/1.1\r\n" +
		"HOST: 239.255.255.250:1900\r\n" +
		"MAN: \"ssdp:discover\"\r\n" +
		"MX: 2\r\n" +
		"ST: " + sonosURN + "\r\n" +
		"\r\n",
)

// Device represents a discovered Sonos device.
type Device struct {
	IP       string    `json:"ip"`
	Port     int       `json:"port"`
	UUID     string    `json:"uuid"`
	Model    string    `json:"model"`
	Name     string    `json:"name"`
	Location string    `json:"location"`
	LastSeen time.Time `json:"last_seen"`
}

// deviceCache is the on-disk cache format.
type deviceCache struct {
	CachedAt time.Time  `json:"cached_at"`
	Devices  []*Device  `json:"devices"`
}

// Discovery handles Sonos device discovery via SSDP.
type Discovery struct {
	timeout  time.Duration
	ttl      time.Duration
	cacheDir string

	mu      sync.RWMutex
	devices map[string]*Device // keyed by UUID
	aliases map[string]string  // alias -> UUID
}

// NewDiscovery creates a new Discovery instance.
func NewDiscovery(timeout time.Duration) *Discovery {
	if timeout == 0 {
		timeout = 3 * time.Second
	}

	// Determine cache directory
	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir == "" {
		home, _ := os.UserHomeDir()
		cacheDir = filepath.Join(home, ".cache")
	}
	cacheDir = filepath.Join(cacheDir, "riff")

	return &Discovery{
		timeout:  timeout,
		ttl:      defaultTTL,
		cacheDir: cacheDir,
		devices:  make(map[string]*Device),
		aliases:  make(map[string]string),
	}
}

// cacheFilePath returns the path to the device cache file.
func (d *Discovery) cacheFilePath() string {
	return filepath.Join(d.cacheDir, "sonos-devices.json")
}

// loadCache reads devices from the file cache.
func (d *Discovery) loadCache() ([]*Device, bool) {
	data, err := os.ReadFile(d.cacheFilePath())
	if err != nil {
		return nil, false
	}

	var cache deviceCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, false
	}

	// Check if cache is still valid
	if time.Since(cache.CachedAt) > fileCacheTTL {
		return nil, false
	}

	// Populate in-memory cache
	d.mu.Lock()
	for _, dev := range cache.Devices {
		d.devices[dev.UUID] = dev
	}
	d.mu.Unlock()

	return cache.Devices, true
}

// saveCache writes devices to the file cache.
func (d *Discovery) saveCache(devices []*Device) {
	cache := deviceCache{
		CachedAt: time.Now(),
		Devices:  devices,
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(d.cacheDir, 0755); err != nil {
		return
	}

	_ = os.WriteFile(d.cacheFilePath(), data, 0644)
}

// SetAlias maps an alias name to a device UUID or IP.
func (d *Discovery) SetAlias(alias, target string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.aliases[strings.ToLower(alias)] = target
}

// Discover performs SSDP discovery and returns all found Sonos devices.
// Results are cached to ~/.cache/riff/sonos-devices.json for faster subsequent lookups.
func (d *Discovery) Discover(ctx context.Context) ([]*Device, error) {
	// Check file cache first
	if devices, ok := d.loadCache(); ok {
		return devices, nil
	}

	return d.discoverSSDP(ctx)
}

// DiscoverFresh bypasses the cache and performs fresh SSDP discovery.
func (d *Discovery) DiscoverFresh(ctx context.Context) ([]*Device, error) {
	return d.discoverSSDP(ctx)
}

// discoverSSDP performs the actual SSDP discovery.
func (d *Discovery) discoverSSDP(ctx context.Context) ([]*Device, error) {
	addr, err := net.ResolveUDPAddr("udp4", ssdpAddr)
	if err != nil {
		return nil, fmt.Errorf("resolve ssdp addr: %w", err)
	}

	conn, err := net.ListenUDP("udp4", nil)
	if err != nil {
		return nil, fmt.Errorf("listen udp: %w", err)
	}
	defer func() { _ = conn.Close() }()

	// Set read deadline
	deadline := time.Now().Add(d.timeout)
	_ = conn.SetReadDeadline(deadline)

	// Send M-SEARCH
	if _, err := conn.WriteToUDP(mSearchRequest, addr); err != nil {
		return nil, fmt.Errorf("send m-search: %w", err)
	}

	// Collect responses
	var devices []*Device
	seen := make(map[string]bool)
	buf := make([]byte, 2048)

	for {
		select {
		case <-ctx.Done():
			d.saveCache(devices)
			return devices, ctx.Err()
		default:
		}

		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				break // Discovery complete
			}
			continue
		}

		device, err := parseResponse(buf[:n], remoteAddr)
		if err != nil || device == nil {
			continue
		}

		if seen[device.UUID] {
			continue
		}
		seen[device.UUID] = true

		device.LastSeen = time.Now()
		devices = append(devices, device)

		// Cache the device in memory
		d.mu.Lock()
		d.devices[device.UUID] = device
		d.mu.Unlock()
	}

	// Save to file cache
	d.saveCache(devices)

	return devices, nil
}

// GetDevice returns a cached device by UUID, name, or alias.
func (d *Discovery) GetDevice(identifier string) *Device {
	d.mu.RLock()
	defer d.mu.RUnlock()

	id := strings.ToLower(identifier)

	// Check aliases first
	if target, ok := d.aliases[id]; ok {
		identifier = target
	}

	// Try by UUID
	if dev, ok := d.devices[identifier]; ok {
		if time.Since(dev.LastSeen) < d.ttl {
			return dev
		}
	}

	// Try by name or IP
	for _, dev := range d.devices {
		if time.Since(dev.LastSeen) >= d.ttl {
			continue
		}
		if strings.EqualFold(dev.Name, identifier) || dev.IP == identifier {
			return dev
		}
	}

	return nil
}

// CachedDevices returns all cached devices that haven't expired.
func (d *Discovery) CachedDevices() []*Device {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var devices []*Device
	now := time.Now()
	for _, dev := range d.devices {
		if now.Sub(dev.LastSeen) < d.ttl {
			devices = append(devices, dev)
		}
	}
	return devices
}

// parseResponse parses an SSDP response into a Device.
func parseResponse(data []byte, addr *net.UDPAddr) (*Device, error) {
	resp, err := http.ReadResponse(bufio.NewReader(strings.NewReader(string(data))), nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	// Verify it's a Sonos device
	st := resp.Header.Get("ST")
	if st != sonosURN {
		return nil, nil
	}

	location := resp.Header.Get("Location")
	usn := resp.Header.Get("USN")

	// Extract UUID from USN (format: uuid:RINCON_xxx::urn:...)
	uuid := extractUUID(usn)
	if uuid == "" {
		return nil, nil
	}

	// Extract port from location URL
	port := 1400 // default Sonos port
	if location != "" {
		if strings.Contains(location, ":") {
			// Parse port from location
			parts := strings.Split(location, ":")
			if len(parts) >= 3 {
				portStr := strings.Split(parts[2], "/")[0]
				_, _ = fmt.Sscanf(portStr, "%d", &port)
			}
		}
	}

	return &Device{
		IP:       addr.IP.String(),
		Port:     port,
		UUID:     uuid,
		Location: location,
	}, nil
}

// extractUUID extracts the UUID from a USN header.
func extractUUID(usn string) string {
	// Format: uuid:RINCON_xxx::urn:schemas-upnp-org:device:ZonePlayer:1
	if !strings.HasPrefix(usn, "uuid:") {
		return ""
	}
	parts := strings.Split(usn, "::")
	if len(parts) < 1 {
		return ""
	}
	return strings.TrimPrefix(parts[0], "uuid:")
}
