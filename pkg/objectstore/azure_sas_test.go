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
	perms := AzureSASPermissions{Read: true}
	key := account + "/" + container + "/" + perms.toContainerPermissions().String()

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
	got, err := MintAzureUserDelegationSAS(context.Background(), account, container, perms)
	if err != nil {
		t.Fatalf("unexpected error on cache hit: %v", err)
	}
	if got != "cached-sas" {
		t.Errorf("got %q, want %q (cache hit)", got, "cached-sas")
	}
}

func TestMintAzureUserDelegationSASCacheKeyPerPermissions(t *testing.T) {
	const account, container = "acct", "cont"
	// A token cached for read-only permissions must not satisfy a request for a
	// different permission set (which would otherwise fall through to a network
	// mint and fail with no Azure identity available).
	readOnly := AzureSASPermissions{Read: true}
	key := account + "/" + container + "/" + readOnly.toContainerPermissions().String()
	azureSASCacheMu.Lock()
	azureSASCache[key] = cachedAzureSAS{token: "read-only-sas", expiry: time.Now().UTC().Add(azureSASValidity)}
	azureSASCacheMu.Unlock()
	t.Cleanup(func() {
		azureSASCacheMu.Lock()
		delete(azureSASCache, key)
		azureSASCacheMu.Unlock()
	})

	if _, err := MintAzureUserDelegationSAS(context.Background(), account, container, AzureSASPermissions{Write: true}); err == nil {
		t.Error("expected a network mint (error) for a permission set not present in the cache")
	}
}

func TestMintAzureUserDelegationSASValidation(t *testing.T) {
	perms := AllAzureSASPermissions()
	if _, err := MintAzureUserDelegationSAS(context.Background(), "", "cont", perms); err == nil {
		t.Error("expected error for empty storage account")
	}
	if _, err := MintAzureUserDelegationSAS(context.Background(), "acct", "", perms); err == nil {
		t.Error("expected error for empty container name")
	}
}
