package objectstore

import (
	"context"
	"testing"
	"time"
)

func TestCachedAzureSASFresh(t *testing.T) {
	now := time.Now().UTC()
	cases := []struct {
		name   string
		expiry time.Time
		want   bool
	}{
		{"well within validity", now.Add(azureSASValidity), true},
		{"inside refresh margin", now.Add(azureSASRefreshMargin - time.Minute), false},
		{"already expired", now.Add(-time.Minute), false},
	}
	for _, tc := range cases {
		if got := (cachedAzureSAS{token: "t", expiry: tc.expiry}).fresh(); got != tc.want {
			t.Errorf("%s: fresh()=%v want %v", tc.name, got, tc.want)
		}
	}
}

func TestMintAzureUserDelegationSASCacheHit(t *testing.T) {
	const account, container = "acct", "cont"
	key := account + "/" + container

	azureSASCacheMu.Lock()
	azureSASCache[key] = cachedAzureSAS{token: "cached-sas", expiry: time.Now().UTC().Add(azureSASValidity)}
	azureSASCacheMu.Unlock()
	t.Cleanup(func() {
		azureSASCacheMu.Lock()
		delete(azureSASCache, key)
		azureSASCacheMu.Unlock()
	})

	// A fresh cache entry must be returned via the fast path without touching
	// the network (which would fail here with no Azure identity available).
	got, err := MintAzureUserDelegationSAS(context.Background(), account, container)
	if err != nil {
		t.Fatalf("unexpected error on cache hit: %v", err)
	}
	if got != "cached-sas" {
		t.Errorf("got %q, want %q (cache hit)", got, "cached-sas")
	}
}

func TestMintAzureUserDelegationSASValidation(t *testing.T) {
	if _, err := MintAzureUserDelegationSAS(context.Background(), "", "cont"); err == nil {
		t.Error("expected error for empty storage account")
	}
	if _, err := MintAzureUserDelegationSAS(context.Background(), "acct", ""); err == nil {
		t.Error("expected error for empty container name")
	}
}
