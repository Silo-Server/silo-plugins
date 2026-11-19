package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Silo-Server/silo-plugins/catalog"
)

func TestPublishedDescriptorIsImmutable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "releases", "silo.auth.oidc", "0.1.2.json")
	pkg := catalog.CatalogPackage{
		Tag:      "v0.1.2",
		Binaries: map[string]catalog.PlatformBinary{"linux/amd64": {Checksum: strings.Repeat("a", 64)}},
	}
	if created, err := publishDescriptor(path, pkg); err != nil || !created {
		t.Fatalf("initial publication = %v, %v", created, err)
	}
	initial, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if created, err := publishDescriptor(path, pkg); err != nil || created {
		t.Fatalf("repeated publication = %v, %v", created, err)
	}
	pkg.Binaries["linux/amd64"] = catalog.PlatformBinary{Checksum: strings.Repeat("b", 64)}
	if created, err := publishDescriptor(path, pkg); err == nil || created {
		t.Fatalf("edited publication = %v, %v", created, err)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(initial, after) {
		t.Fatalf("descriptor changed after a refused edit: %v", err)
	}
}
