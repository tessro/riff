package sonos

import (
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"strings"
)

// Group represents a Sonos speaker group.
type Group struct {
	ID          string   `json:"id"`
	Coordinator *Device  `json:"coordinator"`
	Members     []*Device `json:"members"`
	Name        string   `json:"name"`
}

// ZoneGroupState contains the parsed zone group topology.
type ZoneGroupState struct {
	Groups []Group `json:"groups"`
}

// GetZoneGroupState retrieves the current zone group topology.
func (c *Client) GetZoneGroupState(ctx context.Context, device *Device) (*ZoneGroupState, error) {
	resp, err := c.soap.Call(ctx, device.IP, device.Port, ZoneGroupTopologyEndpoint, ZoneGroupTopologyService, "GetZoneGroupState", nil)
	if err != nil {
		return nil, err
	}

	var envelope struct {
		Body struct {
			Response struct {
				ZoneGroupState string `xml:"ZoneGroupState"`
			} `xml:"GetZoneGroupStateResponse"`
		} `xml:"Body"`
	}
	if err := xml.Unmarshal(resp, &envelope); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return parseZoneGroupState(html.UnescapeString(envelope.Body.Response.ZoneGroupState))
}

// ListGroups returns all speaker groups.
func (c *Client) ListGroups(ctx context.Context, device *Device) ([]Group, error) {
	state, err := c.GetZoneGroupState(ctx, device)
	if err != nil {
		return nil, err
	}
	return state.Groups, nil
}

// AddToGroup adds a device to a group.
func (c *Client) AddToGroup(ctx context.Context, device *Device, coordinatorUUID string) error {
	args := map[string]string{
		"InstanceID":            "0",
		"CurrentURI":            fmt.Sprintf("x-rincon:%s", coordinatorUUID),
		"CurrentURIMetaData":    "",
	}
	_, err := c.soap.Call(ctx, device.IP, device.Port, AVTransportEndpoint, AVTransportService, "SetAVTransportURI", args)
	return err
}

// RemoveFromGroup removes a device from its group (makes it standalone).
func (c *Client) RemoveFromGroup(ctx context.Context, device *Device) error {
	args := map[string]string{
		"InstanceID": "0",
	}
	_, err := c.soap.Call(ctx, device.IP, device.Port, AVTransportEndpoint, AVTransportService, "BecomeCoordinatorOfStandaloneGroup", args)
	return err
}

// parseZoneGroupState parses the XML zone group state.
func parseZoneGroupState(xmlData string) (*ZoneGroupState, error) {
	type ZoneMember struct {
		UUID     string `xml:"UUID,attr"`
		Location string `xml:"Location,attr"`
		ZoneName string `xml:"ZoneName,attr"`
	}

	type ZoneGroup struct {
		Coordinator string       `xml:"Coordinator,attr"`
		ID          string       `xml:"ID,attr"`
		Members     []ZoneMember `xml:"ZoneGroupMember"`
	}

	type ZoneGroups struct {
		Groups []ZoneGroup `xml:"ZoneGroup"`
	}

	type ZoneGroupStateXML struct {
		ZoneGroups ZoneGroups `xml:"ZoneGroups"`
	}

	var state ZoneGroupStateXML
	if err := xml.Unmarshal([]byte(xmlData), &state); err != nil {
		return nil, fmt.Errorf("parse zone group state: %w", err)
	}

	result := &ZoneGroupState{}
	for _, zg := range state.ZoneGroups.Groups {
		group := Group{
			ID: zg.ID,
		}

		for _, m := range zg.Members {
			dev := &Device{
				UUID: m.UUID,
				Name: m.ZoneName,
			}
			// Extract IP from location
			if m.Location != "" {
				parts := strings.Split(m.Location, "//")
				if len(parts) > 1 {
					hostPort := strings.Split(parts[1], "/")[0]
					dev.IP = strings.Split(hostPort, ":")[0]
				}
			}

			if m.UUID == zg.Coordinator {
				group.Coordinator = dev
				group.Name = m.ZoneName
			}
			group.Members = append(group.Members, dev)
		}

		result.Groups = append(result.Groups, group)
	}

	return result, nil
}
