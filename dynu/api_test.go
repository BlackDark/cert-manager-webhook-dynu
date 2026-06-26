package dynu

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func setupTestServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	oldBaseURL := baseURL
	baseURL = srv.URL
	t.Cleanup(func() {
		srv.Close()
		baseURL = oldBaseURL
	})
}

func TestResolveZone(t *testing.T) {
	setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method %s", r.Method)
		}
		switch {
		case strings.HasPrefix(r.URL.Path, "/dns/getroot/example.com"):
			json.NewEncoder(w).Encode(dnsRootResponse{ID: 42, DomainName: "example.com", Node: "_acme-challenge"})
		default:
			http.NotFound(w, r)
		}
	})

	api := &API{APIKey: "test-key"}
	domainID, node, err := api.ResolveZone(context.Background(), "_acme-challenge.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if domainID != 42 {
		t.Fatalf("domainID = %d, want 42", domainID)
	}
	if node != "_acme-challenge" {
		t.Fatalf("node = %q, want %q", node, "_acme-challenge")
	}
}

func TestAddTXTRecord_returnsAdded_whenNewRecord(t *testing.T) {
	setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			json.NewEncoder(w).Encode(dnsRecordsResponse{})
		case http.MethodPost:
			w.WriteHeader(http.StatusOK)
		}
	})

	api := &API{APIKey: "test-key"}
	added, err := api.AddTXTRecord(context.Background(), 42, "_acme-challenge", "somekey")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !added {
		t.Fatal("expected added=true for new record")
	}
}

func TestAddTXTRecord_returnsNotAdded_whenRecordAlreadyExists(t *testing.T) {
	existing := dnsRecordsResponse{
		DNSRecords: []DNSRecord{{
			ID: 1, DomainID: 42, NodeName: "_acme-challenge", RecordType: "TXT", TextData: "somekey",
		}},
	}
	setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(existing)
		}
	})

	api := &API{APIKey: "test-key"}
	added, err := api.AddTXTRecord(context.Background(), 42, "_acme-challenge", "somekey")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if added {
		t.Fatal("expected added=false when record already exists")
	}
}

func TestDeleteTXTRecord_returnsDeleted_whenRecordExists(t *testing.T) {
	existing := dnsRecordsResponse{
		DNSRecords: []DNSRecord{{
			ID: 7, DomainID: 42, NodeName: "_acme-challenge", RecordType: "TXT", TextData: "somekey",
		}},
	}
	setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			json.NewEncoder(w).Encode(existing)
		case http.MethodDelete:
			w.WriteHeader(http.StatusOK)
		}
	})

	api := &API{APIKey: "test-key"}
	deleted, err := api.DeleteTXTRecord(context.Background(), 42, "somekey")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Fatal("expected deleted=true when record was found and removed")
	}
}

func TestDeleteTXTRecord_returnsNotDeleted_whenRecordNotFound(t *testing.T) {
	setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(dnsRecordsResponse{})
		}
	})

	api := &API{APIKey: "test-key"}
	deleted, err := api.DeleteTXTRecord(context.Background(), 42, "somekey")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleted {
		t.Fatal("expected deleted=false when record not found")
	}
}
