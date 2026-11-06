package layer4

import (
	"crypto/tls"
	"crypto/x509"
	"testing"
)

func TestEmbeddedISRGRootParses(t *testing.T) {
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM([]byte(isrgRootX1)) {
		t.Fatal("embedded ISRG Root X1 did not parse")
	}
}

func TestRegistryVerifiesWithISRGRootOnly(t *testing.T) {
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM([]byte(isrgRootX1)) {
		t.Fatal("embedded ISRG Root X1 did not parse")
	}
	conn, err := tls.Dial("tcp", "clio.themobileprof.com:443", &tls.Config{
		ServerName: "clio.themobileprof.com",
		RootCAs:    pool,
		MinVersion: tls.VersionTLS12,
	})
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
}

func TestEmptyPoolRejectsRegistry(t *testing.T) {
	_, err := tls.Dial("tcp", "clio.themobileprof.com:443", &tls.Config{
		ServerName: "clio.themobileprof.com",
		RootCAs:    x509.NewCertPool(),
		MinVersion: tls.VersionTLS12,
	})
	if err == nil {
		t.Fatal("empty trust store accepted the registry certificate")
	}
}
