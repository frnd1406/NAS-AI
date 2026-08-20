package content_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/nas-ai/api/src/drivers/storage"
	"github.com/nas-ai/api/src/services/content"
)

func TestUserHomeIsolation(t *testing.T) {
	tmp := t.TempDir()
	store, err := storage.NewLocalStore(tmp)
	if err != nil {
		t.Fatal(err)
	}
	mgr := content.NewStorageManager(store, nil, nil, nil)
	uidA := "11111111-1111-1111-1111-111111111111"
	uidB := "22222222-2222-2222-2222-222222222222"

	a, err := mgr.ForUser(uidA)
	if err != nil {
		t.Fatal(err)
	}
	if err := mgr.EnsureUserHome(uidA); err != nil {
		t.Fatal(err)
	}
	b, err := mgr.ForUser(uidB)
	if err != nil {
		t.Fatal(err)
	}
	if err := mgr.EnsureUserHome(uidB); err != nil {
		t.Fatal(err)
	}

	full, err := a.GetFullPath("Fotos/iPhone")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(filepath.ToSlash(full), "/homes/"+uidA+"/Fotos/iPhone") &&
		!strings.Contains(filepath.ToSlash(full), "\\homes\\"+uidA+"\\Fotos\\iPhone") &&
		!strings.HasSuffix(filepath.ToSlash(full), "homes/"+uidA+"/Fotos/iPhone") {
		t.Fatalf("path not under user home: %s", full)
	}

	items, err := a.List("")
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range items {
		switch it.Name {
		case "api", "tls", "dev-certs", "homes":
			t.Fatalf("dangerous entry visible in home: %s", it.Name)
		}
	}

	if err := a.Mkdir("private-only"); err != nil {
		t.Fatal(err)
	}
	otherItems, err := b.List("")
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range otherItems {
		if it.Name == "private-only" {
			t.Fatal("user B can see user A private folder")
		}
	}
}
