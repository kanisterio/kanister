package objectstore

import (
	"context"
	"testing"
)

func TestMintAzureUserDelegationSASValidation(t *testing.T) {
	perms := AllAzureSASPermissions()
	if _, err := MintAzureUserDelegationSAS(context.Background(), "", "cont", perms); err == nil {
		t.Error("expected error for empty storage account")
	}
	if _, err := MintAzureUserDelegationSAS(context.Background(), "acct", "", perms); err == nil {
		t.Error("expected error for empty container name")
	}
}
