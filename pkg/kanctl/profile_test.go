package kanctl

import (
	check "gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/secrets"
)

func (k *KanctlTestSuite) TestConstructProfile(c *check.C) {
	for _, tc := range []struct {
		lp          *locationParams
		secret      *corev1.Secret
		retCredType crv1alpha1.CredentialType
	}{
		{
			lp: &locationParams{
				locationType: crv1alpha1.LocationTypeS3Compliant,
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
				StringData: map[string]string{
					secrets.ConfigRole: "role",
				},
			},
			retCredType: crv1alpha1.CredentialTypeSecret,
		},
		{
			lp: &locationParams{
				locationType: crv1alpha1.LocationTypeAzure,
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
				StringData: map[string]string{
					secrets.AzureStorageEnvironment: "env",
				},
			},
			retCredType: crv1alpha1.CredentialTypeSecret,
		},
		{
			lp: &locationParams{
				locationType: crv1alpha1.LocationTypeAzure,
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
				StringData: map[string]string{
					secrets.AzureStorageEnvironment: "",
				},
			},
			retCredType: crv1alpha1.CredentialTypeKeyPair,
		},
		{
			lp: &locationParams{
				locationType: crv1alpha1.LocationTypeAzure,
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
				StringData: map[string]string{},
			},
			retCredType: crv1alpha1.CredentialTypeKeyPair,
		},
		{
			lp: &locationParams{
				locationType: crv1alpha1.LocationTypeGCS,
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
			},
			retCredType: crv1alpha1.CredentialTypeKeyPair,
		},
		{
			lp: &locationParams{
				locationType: crv1alpha1.LocationTypeS3Compliant,
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
			},
			retCredType: crv1alpha1.CredentialTypeKeyPair,
		},
	} {
		prof := constructProfile(tc.lp, tc.secret)
		c.Assert(prof.Credential.Type, check.Equals, tc.retCredType)
	}
}
