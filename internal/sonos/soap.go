package sonos

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// UPnP service endpoints
	AVTransportEndpoint      = "/MediaRenderer/AVTransport/Control"
	RenderingControlEndpoint = "/MediaRenderer/RenderingControl/Control"
	ZoneGroupTopologyEndpoint = "/ZoneGroupTopology/Control"
	DevicePropertiesEndpoint = "/DeviceProperties/Control"

	// UPnP service URNs
	AVTransportService      = "urn:schemas-upnp-org:service:AVTransport:1"
	RenderingControlService = "urn:schemas-upnp-org:service:RenderingControl:1"
	ZoneGroupTopologyService = "urn:upnp-org:serviceId:ZoneGroupTopology"
	DevicePropertiesService = "urn:upnp-org:serviceId:DeviceProperties"
)

// SOAPClient makes SOAP requests to Sonos devices.
type SOAPClient struct {
	httpClient *http.Client
}

// NewSOAPClient creates a new SOAP client.
func NewSOAPClient() *SOAPClient {
	return &SOAPClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SOAPEnvelope wraps a SOAP request/response.
type SOAPEnvelope struct {
	XMLName xml.Name `xml:"s:Envelope"`
	NS      string   `xml:"xmlns:s,attr"`
	Body    SOAPBody `xml:"s:Body"`
}

// SOAPBody contains the SOAP body content.
type SOAPBody struct {
	Content []byte `xml:",innerxml"`
}

// Call makes a SOAP request to a Sonos device.
func (c *SOAPClient) Call(ctx context.Context, host string, port int, endpoint, service, action string, args map[string]string) ([]byte, error) {
	url := fmt.Sprintf("http://%s:%d%s", host, port, endpoint)

	// Build SOAP body
	body := c.buildSOAPBody(service, action, args)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", fmt.Sprintf("\"%s#%s\"", service, action))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("soap request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("soap error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// buildSOAPBody constructs the SOAP envelope.
func (c *SOAPClient) buildSOAPBody(service, action string, args map[string]string) []byte {
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="utf-8"?>`)
	buf.WriteString(`<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">`)
	buf.WriteString(`<s:Body>`)
	buf.WriteString(fmt.Sprintf(`<u:%s xmlns:u="%s">`, action, service))

	for k, v := range args {
		buf.WriteString(fmt.Sprintf("<%s>%s</%s>", k, xmlEscape(v), k))
	}

	buf.WriteString(fmt.Sprintf(`</u:%s>`, action))
	buf.WriteString(`</s:Body>`)
	buf.WriteString(`</s:Envelope>`)

	return buf.Bytes()
}

// xmlEscape escapes special XML characters.
func xmlEscape(s string) string {
	var buf bytes.Buffer
	xml.EscapeText(&buf, []byte(s))
	return buf.String()
}
