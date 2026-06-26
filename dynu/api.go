package dynu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultTTL = 60

var (
	baseURL       = "https://api.dynu.com/v2"
	defaultClient = &http.Client{Timeout: 30 * time.Second}
)

// API is a client for the Dynu DNS API v2.
type API struct {
	APIKey string
}

type dnsRootResponse struct {
	ID         int    `json:"id"`
	DomainName string `json:"domainName"`
	Node       string `json:"node"`
}

type domain struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type domainsResponse struct {
	Domains []domain `json:"domains"`
}

// DNSRecord represents a Dynu DNS record.
type DNSRecord struct {
	ID         int    `json:"id"`
	DomainID   int    `json:"domainId"`
	NodeName   string `json:"nodeName"`
	RecordType string `json:"recordType"`
	TextData   string `json:"textData"`
}

type dnsRecordsResponse struct {
	DNSRecords []DNSRecord `json:"dnsRecords"`
}

type apiError struct {
	StatusCode int    `json:"statusCode"`
	Type       string `json:"type"`
	Message    string `json:"message"`
}

func (a *API) request(ctx context.Context, method, path string, body io.Reader, target any) error {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("API-Key", a.APIKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := defaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		var errResp apiError
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return fmt.Errorf("%s %s: HTTP %d", method, path, resp.StatusCode)
		}
		if errResp.Message != "" {
			return fmt.Errorf("%s %s: %s", method, path, errResp.Message)
		}
		return fmt.Errorf("%s %s: HTTP %d", method, path, resp.StatusCode)
	}

	if target != nil {
		if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
			return fmt.Errorf("%s %s response parsing error: %w", method, path, err)
		}
	}

	return nil
}

func (a *API) getRoot(ctx context.Context, hostname string) (*dnsRootResponse, error) {
	root := new(dnsRootResponse)
	if err := a.request(ctx, http.MethodGet, "/dns/getroot/"+hostname, nil, root); err != nil {
		return nil, err
	}
	return root, nil
}

func (a *API) listDomains(ctx context.Context) ([]domain, error) {
	resp := new(domainsResponse)
	if err := a.request(ctx, http.MethodGet, "/dns", nil, resp); err != nil {
		return nil, err
	}
	return resp.Domains, nil
}

func (a *API) listRecords(ctx context.Context, domainID int) ([]DNSRecord, error) {
	resp := new(dnsRecordsResponse)
	path := fmt.Sprintf("/dns/%d/record", domainID)
	if err := a.request(ctx, http.MethodGet, path, nil, resp); err != nil {
		return nil, err
	}
	return resp.DNSRecords, nil
}

// ResolveZone maps an ACME challenge FQDN to a Dynu domain ID and node name.
func (a *API) ResolveZone(ctx context.Context, fqdn string) (domainID int, nodeName string, err error) {
	fqdn = strings.TrimSuffix(fqdn, ".")
	parts := strings.Split(fqdn, ".")
	if len(parts) < 2 {
		return 0, "", fmt.Errorf("invalid fqdn %q", fqdn)
	}

	for i := 2; i <= len(parts); i++ {
		candidate := strings.Join(parts[i-1:], ".")
		root, err := a.getRoot(ctx, candidate)
		if err != nil {
			continue
		}
		if root.DomainName != candidate {
			continue
		}

		p := i - 1
		node := strings.Join(parts[:p], ".")
		if strings.Contains(node, ".") {
			if subID, subNode, ok := a.matchSubdomainZone(ctx, fqdn); ok {
				return subID, subNode, nil
			}
		}
		return root.ID, node, nil
	}

	return 0, "", fmt.Errorf("no Dynu zone found for fqdn %q", fqdn)
}

func (a *API) matchSubdomainZone(ctx context.Context, fqdn string) (int, string, bool) {
	domains, err := a.listDomains(ctx)
	if err != nil {
		return 0, "", false
	}

	fqdn = strings.TrimSuffix(fqdn, ".")
	parts := strings.Split(fqdn, ".")
	for i := range parts {
		candidate := strings.Join(parts[i:], ".")
		for _, d := range domains {
			if d.Name != candidate {
				continue
			}
			node := strings.TrimSuffix(fqdn, "."+d.Name)
			return d.ID, node, true
		}
	}
	return 0, "", false
}

// AddTXTRecord creates a TXT record when it does not already exist.
// Returns added=true when a new record was created.
func (a *API) AddTXTRecord(ctx context.Context, domainID int, nodeName, text string) (bool, error) {
	records, err := a.listRecords(ctx, domainID)
	if err != nil {
		return false, err
	}

	for _, record := range records {
		if record.RecordType == "TXT" && record.NodeName == nodeName && record.TextData == text {
			return false, nil
		}
	}

	payload := map[string]any{
		"nodeName":   nodeName,
		"recordType": "TXT",
		"ttl":        defaultTTL,
		"state":      true,
		"textData":   text,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return false, err
	}

	path := fmt.Sprintf("/dns/%d/record", domainID)
	if err := a.request(ctx, http.MethodPost, path, bytes.NewReader(raw), nil); err != nil {
		return false, err
	}
	return true, nil
}

// DeleteTXTRecord removes TXT records that match text exactly.
// Returns deleted=true when at least one matching record was removed.
func (a *API) DeleteTXTRecord(ctx context.Context, domainID int, text string) (bool, error) {
	records, err := a.listRecords(ctx, domainID)
	if err != nil {
		return false, err
	}

	deleted := false
	for _, record := range records {
		if record.RecordType != "TXT" || record.TextData != text {
			continue
		}
		path := fmt.Sprintf("/dns/%d/record/%d", domainID, record.ID)
		if err := a.request(ctx, http.MethodDelete, path, nil, nil); err != nil {
			return deleted, err
		}
		deleted = true
	}
	return deleted, nil
}
