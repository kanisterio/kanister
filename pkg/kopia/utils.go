package kopia

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// defaultConfigFilePath is the file which contains kopia repo config
	defaultConfigFilePath = "/tmp/kopia-repository.config"

	// defaultCacheDirectory is the directory where kopia content cache is created
	defaultCacheDirectory = "/tmp/kopia-cache"

	// defaultDataStoreGeneralContentCacheSizeMB is the default content cache size for general command workloads
	defaultDataStoreGeneralContentCacheSizeMB = 0

	// defaultDataStoreGeneralMetadataCacheSizeMB is the default metadata cache size for general command workloads
	defaultDataStoreGeneralMetadataCacheSizeMB = 500

	// tlsCertificateKey represents the key used to fetch the certificate
	// from the secret.
	tlsCertificateKey = "tls.crt"
)

// ExtractFingerprintFromCertSecret extracts the fingerprint from the given certificate secret
func ExtractFingerprintFromCertSecret(ctx context.Context, cli kubernetes.Interface, secretName, secretNamespace string) (string, error) {
	secret, err := cli.CoreV1().Secrets(secretNamespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrapf(err, "Failed to get Certificate Secret. Secret: %s", secretName)
	}

	certBytes, err := json.Marshal(secret.Data[tlsCertificateKey])
	if err != nil {
		return "", errors.Wrap(err, "Failed to marshal Certificate Secret Data")
	}

	var certString string
	if err := json.Unmarshal([]byte(certBytes), &certString); err != nil {
		return "", errors.Wrap(err, "Failed to unmarshal Certificate Secret Data")
	}

	decodedCertData, err := base64.StdEncoding.DecodeString(certString)
	if err != nil {
		return "", errors.Wrap(err, "Failed to decode Certificate Secret Data")
	}

	return extractFingerprintFromSliceOfBytes(decodedCertData)
}

// extractFingerprintFromSliceOfBytes extracts the fingeprint from the
// certificate data provided in slice of bytes (default type for secret.Data)
func extractFingerprintFromSliceOfBytes(pemData []byte) (string, error) {
	block, rest := pem.Decode([]byte(pemData))
	if block == nil || len(rest) > 0 {
		return "", errors.New("Failed to PEM Decode Kopia API Server Certificate Secret Data")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", errors.Wrap(err, "Failed to parse X509 Kopia API Server Certificate Secret Data")
	}

	fingerprint := sha256.Sum256(cert.Raw)
	return hex.EncodeToString(fingerprint[:]), nil
}
