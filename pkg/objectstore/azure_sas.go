package objectstore

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
	"github.com/kanisterio/errkit"
)

const (
	// azureSASValidity is the lifetime of a minted user-delegation SAS token.
	//
	// The token is minted per operation and injected into a single worker pod
	// (there is no cross-pod cache — see MintAzureUserDelegationSAS), so its
	// lifetime is the blast-radius bound for exactly that one pod. A
	// user-delegation SAS cannot be revoked individually, so this validity is
	// the only such bound.
	//
	// It must outlast the longest single kando operation (e.g. a `location push`
	// of a very large logical export) or the SAS would expire mid-upload.
	// Worst-case sizing — a multi-terabyte single-object export at a
	// conservative sustained upload throughput of ~50 MB/s: 1 TiB ≈ 5.8h,
	// 5 TiB ≈ 29h. Adding margin for retries/throttling/TLS/variance and
	// rounding up gives 48h, which covers ~8 TiB @50 MB/s (~4 TiB even @25 MB/s)
	// and stays well under the 7-day user-delegation-key ceiling. Raise it (to
	// <7d) if a larger single-object export must be supported.
	azureSASValidity = 48 * time.Hour
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

// MintAzureUserDelegationSAS returns a short-lived, container-scoped Azure
// user-delegation SAS token for the given storage account and container,
// minted using the pod's Azure Workload (federated) Identity.
//
// It mints a fresh token on every call — there is deliberately no cross-call
// cache. The injector runs in kanister-svc and builds worker pods across many
// namespaces; a process-level cache would hand one long-lived token to all of
// them (a token that, given the TB-scale validity, can outlive the pod that
// read it). Minting per invocation binds each token to a single pod's operation
// and keeps its blast radius to that one pod. (The GetUserDelegationCredential
// call this adds per pod-build is one network round-trip; a future optimization
// could cache the user-delegation *key* — safe to reuse for local signing — and
// still sign a distinct per-pod SAS.)
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
	token, _, err := mintAzureUserDelegationSAS(ctx, account, containerName, permStr)
	if err != nil {
		return "", err
	}
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
		Protocol:      sas.ProtocolHTTPS,
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
