package kanctl

import (
	"github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/secrets"
	check "gopkg.in/check.v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (k *KanctlTestSuite) TestConstructProfile(c *check.C) {
	for _, tc := range []struct {
		lp          *locationParams
		secret      *v1.Secret
		retCredType v1alpha1.CredentialType
	}{
		{
			lp: &locationParams{
				locationType: v1alpha1.LocationTypeS3Compliant,
			},
			secret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
				StringData: map[string]string{
					secrets.ConfigRole: "role",
				},
			},
			retCredType: v1alpha1.CredentialTypeSecret,
		},
		{
			lp: &locationParams{
				locationType: v1alpha1.LocationTypeAzure,
			},
			secret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
				StringData: map[string]string{
					secrets.AzureStorageEnvironment: "env",
				},
			},
			retCredType: v1alpha1.CredentialTypeSecret,
		},
		{
			lp: &locationParams{
				locationType: v1alpha1.LocationTypeAzure,
			},
			secret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
				StringData: map[string]string{
					secrets.AzureStorageEnvironment: "",
				},
			},
			retCredType: v1alpha1.CredentialTypeKeyPair,
		},
		{
			lp: &locationParams{
				locationType: v1alpha1.LocationTypeAzure,
			},
			secret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
				StringData: map[string]string{},
			},
			retCredType: v1alpha1.CredentialTypeKeyPair,
		},
		{
			lp: &locationParams{
				locationType: v1alpha1.LocationTypeGCS,
			},
			secret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
			},
			retCredType: v1alpha1.CredentialTypeKeyPair,
		},
		{
			lp: &locationParams{
				locationType: v1alpha1.LocationTypeS3Compliant,
			},
			secret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
			},
			retCredType: v1alpha1.CredentialTypeKeyPair,
		},
	} {
		prof := constructProfile(tc.lp, tc.secret)
		c.Assert(prof.Credential.Type, check.Equals, tc.retCredType)
	}
}
