package objectstore

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
	"github.com/kanisterio/errkit"
)

const (
	// azureSASValidity is the lifetime of a minted user-delegation SAS token.
	azureSASValidity = 24 * time.Hour
	// azureSASRefreshMargin is how long before expiry a cached token is
	// considered stale and re-minted, so a handed-out SAS always has enough
	// remaining validity to outlast an in-flight operation.
	azureSASRefreshMargin = 1 * time.Hour
)

// AzureSASPermissions is the least-privilege permission set granted to a minted
// user-delegation SAS. The zero value grants no permissions; callers should set
// only the permissions the operation needs (e.g. Read for pull, Create+Write+Add
// for push, Delete+List for retire).
type AzureSASPermissions struct {
	Read   bool
	Add    bool
	Create bool
	Write  bool
	Delete bool
	List   bool
}

func (p AzureSASPermissions) toContainerPermissions() *sas.ContainerPermissions {
	return &sas.ContainerPermissions{
		Read:   p.Read,
		Add:    p.Add,
		Create: p.Create,
		Write:  p.Write,
		Delete: p.Delete,
		List:   p.List,
	}
}

// AllAzureSASPermissions grants every container-scoped permission. It is the
// safe default when the specific object-store operation cannot be determined;
// callers that know the operation pass a narrower, least-privilege set.
func AllAzureSASPermissions() AzureSASPermissions {
	return AzureSASPermissions{Read: true, Add: true, Create: true, Write: true, Delete: true, List: true}
}

// cachedAzureSAS is a minted SAS token together with the time it expires.
type cachedAzureSAS struct {
	token  string
	expiry time.Time
}

func (c cachedAzureSAS) fresh() bool {
	return time.Now().UTC().Before(c.expiry.Add(-azureSASRefreshMargin))
}

// azureSASCache caches minted user-delegation SAS tokens for the lifetime of the
// process (e.g. a kando pod), keyed by "<account>/<container>". This avoids a
// fresh workload-identity token acquisition + GetUserDelegationCredential call
// on every object-store operation. Entries are re-minted once they fall within
// azureSASRefreshMargin of expiry.
var (
	azureSASCacheMu sync.Mutex
	azureSASCache   = map[string]cachedAzureSAS{}
)

// MintAzureUserDelegationSAS returns a short-lived, container-scoped Azure
// user-delegation SAS token for the given storage account and container,
// minted using the pod's Azure Workload (federated) Identity. Results are cached
// per account/container for the process lifetime (see azureSASCache); a cached
// token is reused until it is within azureSASRefreshMargin of expiry.
//
// It is used on the non-Kopia object-store data path (e.g. `kando location
// push/pull`) when the storage account has no shared key.
//
// Prerequisites:
//   - The pod must have the workload-identity token injected
//     (AZURE_FEDERATED_TOKEN_FILE/AZURE_CLIENT_ID/AZURE_TENANT_ID) — consumed
//     automatically by NewDefaultAzureCredential.
//   - The managed identity must be able to fetch a user-delegation key on the
//     account (e.g. the "Storage Blob Data Contributor" role).
//
// NOTE: a user-delegation SAS is scoped to a single container and cannot list,
// create, or delete containers — consumers must operate within the given
// container only.
func MintAzureUserDelegationSAS(ctx context.Context, account, containerName string, perms AzureSASPermissions) (string, error) {
	if account == "" {
		return "", errkit.New("Azure storage account is required to mint a SAS token")
	}
	if containerName == "" {
		return "", errkit.New("container name is required to scope the user-delegation SAS token")
	}
	permStr := perms.toContainerPermissions().String()
	// Permissions are part of the cache key: tokens minted for different
	// permission sets must not be shared.
	cacheKey := account + "/" + containerName + "/" + permStr

	// Fast path: reuse a cached token that still has ample validity left. The
	// network mint below is intentionally not held under the lock.
	azureSASCacheMu.Lock()
	if cached, ok := azureSASCache[cacheKey]; ok && cached.fresh() {
		azureSASCacheMu.Unlock()
		return cached.token, nil
	}
	azureSASCacheMu.Unlock()

	token, expiry, err := mintAzureUserDelegationSAS(ctx, account, containerName, permStr)
	if err != nil {
		return "", err
	}

	azureSASCacheMu.Lock()
	azureSASCache[cacheKey] = cachedAzureSAS{token: token, expiry: expiry}
	azureSASCacheMu.Unlock()
	return token, nil
}

// mintAzureUserDelegationSAS performs the actual (network-touching) mint and
// returns the SAS token plus its expiry time.
//
// TODO: sovereign/government cloud endpoints are not handled yet; this assumes
// the public Azure cloud (blob.core.windows.net).
func mintAzureUserDelegationSAS(ctx context.Context, account, containerName, permissions string) (string, time.Time, error) {
	accountURL := fmt.Sprintf("https://%s.blob.core.windows.net/", account)

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return "", time.Time{}, errkit.Wrap(err, "Failed to create Azure workload-identity credential")
	}
	svcClient, err := service.NewClient(accountURL, cred, nil)
	if err != nil {
		return "", time.Time{}, errkit.Wrap(err, "Failed to create Azure blob service client")
	}

	now := time.Now().UTC().Add(-10 * time.Second)
	expiry := now.Add(azureSASValidity)
	info := service.KeyInfo{
		Start:  to.Ptr(now.Format(sas.TimeFormat)),
		Expiry: to.Ptr(expiry.Format(sas.TimeFormat)),
	}
	udc, err := svcClient.GetUserDelegationCredential(ctx, info, nil)
	if err != nil {
		return "", time.Time{}, errkit.Wrap(err, "Failed to get Azure user-delegation key")
	}

	sasQueryParams, err := sas.BlobSignatureValues{
		Protocol:   sas.ProtocolHTTPS,
		StartTime:     now,
		ExpiryTime:    expiry,
		Permissions:   permissions,
		ContainerName: containerName,
	}.SignWithUserDelegation(udc)
	if err != nil {
		return "", time.Time{}, errkit.Wrap(err, "Failed to sign Azure user-delegation SAS token")
	}
	return sasQueryParams.Encode(), expiry, nil
}
