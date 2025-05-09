/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package vclib_test

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/vmware/govmomi/vim25/soap"

	"k8s.io/cloud-provider-vsphere/pkg/common/vclib"
	"k8s.io/cloud-provider-vsphere/pkg/common/vclib/fixtures"
)

func createTestServer(
	t *testing.T,
	caCertPath string,
	serverCertPath string,
	serverKeyPath string,
	handler http.HandlerFunc,
) (*httptest.Server, string) {
	caCertPEM, err := os.ReadFile(caCertPath)
	if err != nil {
		t.Fatalf("Could not read ca cert from file")
	}

	serverCert, err := tls.LoadX509KeyPair(serverCertPath, serverKeyPath)
	if err != nil {
		t.Fatalf("Could not load server cert and server key from files: %#v", err)
	}

	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM(caCertPEM); !ok {
		t.Fatalf("Cannot add CA to CAPool")
	}

	server := httptest.NewUnstartedServer(http.HandlerFunc(handler))
	server.TLS = &tls.Config{
		Certificates: []tls.Certificate{
			serverCert,
		},
		RootCAs: certPool,
	}

	// calculate the leaf certificate's fingerprint
	if len(server.TLS.Certificates) < 1 || len(server.TLS.Certificates[0].Certificate) < 1 {
		t.Fatal("Expected server.TLS.Certificates not to be empty")
	}
	x509LeafCert := server.TLS.Certificates[0].Certificate[0]
	var tpString string
	for i, b := range sha256.Sum256(x509LeafCert) {
		if i > 0 {
			tpString += ":"
		}
		tpString += fmt.Sprintf("%02X", b)
	}

	return server, tpString
}

func TestWithValidCaCert(t *testing.T) {
	handler, verifyConnectionWasMade := getRequestVerifier(t)

	server, _ := createTestServer(t, fixtures.CaCertPath, fixtures.ServerCertPath, fixtures.ServerKeyPath, handler)
	server.StartTLS()
	u := mustParseUrl(t, server.URL)

	connection := &vclib.VSphereConnection{
		Hostname: u.Hostname(),
		Port:     u.Port(),
		CACert:   fixtures.CaCertPath,
	}

	// Ignoring error here, because we only care about the TLS connection
	connection.NewClient(context.Background())

	verifyConnectionWasMade()
}

func TestWithVerificationWithWrongThumbprint(t *testing.T) {
	handler, _ := getRequestVerifier(t)

	server, _ := createTestServer(t, fixtures.CaCertPath, fixtures.ServerCertPath, fixtures.ServerKeyPath, handler)
	server.StartTLS()
	u := mustParseUrl(t, server.URL)

	connection := &vclib.VSphereConnection{
		Hostname:   u.Hostname(),
		Port:       u.Port(),
		Thumbprint: "obviously wrong",
	}

	_, err := connection.NewClient(context.Background())

	if msg := err.Error(); !strings.Contains(msg, "thumbprint does not match") {
		t.Fatalf("Expected wrong thumbprint error, got '%s'", msg)
	}
}

func TestWithVerificationWithoutCaCertOrThumbprint(t *testing.T) {
	handler, _ := getRequestVerifier(t)

	server, _ := createTestServer(t, fixtures.CaCertPath, fixtures.ServerCertPath, fixtures.ServerKeyPath, handler)
	server.StartTLS()
	u := mustParseUrl(t, server.URL)

	connection := &vclib.VSphereConnection{
		Hostname: u.Hostname(),
		Port:     u.Port(),
	}

	_, err := connection.NewClient(context.Background())

	if !soap.IsCertificateUntrusted(err) {
		t.Fatalf("Expected soap.IsCertificateUntrusted, got: '%s' (%#v)", err.Error(), err)
	}
}

func TestWithValidThumbprint(t *testing.T) {
	handler, verifyConnectionWasMade := getRequestVerifier(t)

	server, thumbprint :=
		createTestServer(t, fixtures.CaCertPath, fixtures.ServerCertPath, fixtures.ServerKeyPath, handler)
	server.StartTLS()
	u := mustParseUrl(t, server.URL)

	connection := &vclib.VSphereConnection{
		Hostname:   u.Hostname(),
		Port:       u.Port(),
		Thumbprint: thumbprint,
	}

	// Ignoring error here, because we only care about the TLS connection
	connection.NewClient(context.Background())

	verifyConnectionWasMade()
}

func TestWithInvalidCaCertPath(t *testing.T) {
	connection := &vclib.VSphereConnection{
		Hostname: "should-not-matter",
		Port:     "27015", // doesn't matter, but has to be a valid port
		CACert:   "invalid-path",
	}

	_, err := connection.NewClient(context.Background())
	if _, ok := err.(*os.PathError); !ok {
		t.Fatalf("Expected an os.PathError, got: '%s' (%#v)", err.Error(), err)
	}
}

func TestInvalidCaCert(t *testing.T) {
	connection := &vclib.VSphereConnection{
		Hostname: "should-not-matter",
		Port:     "27015", // doesn't matter, but has to be a valid port
		CACert:   fixtures.InvalidCertPath,
	}

	_, err := connection.NewClient(context.Background())

	if msg := err.Error(); !strings.Contains(msg, "invalid certificate") {
		t.Fatalf("Expected invalid certificate error, got '%s'", msg)
	}
}

func getRequestVerifier(t *testing.T) (http.HandlerFunc, func()) {
	gotRequest := false

	handler := func(w http.ResponseWriter, r *http.Request) {
		gotRequest = true
	}

	checker := func() {
		if !gotRequest {
			t.Fatalf("Never saw a request, maybe TLS connection could not be established?")
		}
	}

	return handler, checker
}

func mustParseUrl(t *testing.T, i string) *url.URL {
	u, err := url.Parse(i)
	if err != nil {
		t.Fatalf("Cannot parse URL: %v", err)
	}
	return u
}
