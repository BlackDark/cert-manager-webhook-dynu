//go:build integration

package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"testing"
	"time"

	acmetest "github.com/cert-manager/cert-manager/test/acme"
)

var zone = os.Getenv("TEST_ZONE_NAME")

func TestRunsSuite(t *testing.T) {
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		t.Fatalf("Failed to generate random challenge key: %v", err)
	}
	randomKey := base64.RawURLEncoding.EncodeToString(randomBytes)
	randomSuffix := hex.EncodeToString(randomBytes[:8])

	fixture := acmetest.NewFixture(&dynuDNSProviderSolver{},
		acmetest.SetResolvedZone(zone),
		acmetest.SetAllowAmbientCredentials(false),
		acmetest.SetManifestPath("testdata/dynu"),
		acmetest.SetDNSChallengeKey(randomKey),
		acmetest.SetResolvedFQDN(fmt.Sprintf("cert-manager-dns01-tests-%s.%s", randomSuffix, zone)),
		acmetest.SetPropagationLimit(time.Minute*15),
	)

	fixture.RunBasic(t)
	fixture.RunExtended(t)
}
