package validate

import (
	"context"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/objectstore"
	"github.com/kanisterio/kanister/pkg/param"
)

// ActionSet function validates the ActionSet and returns an error if it is invalid.
func ActionSet(as *crv1alpha1.ActionSet) error {
	if err := actionSetSpec(as.Spec); err != nil {
		return err
	}
	if as.Status != nil {
		if len(as.Spec.Actions) != len(as.Status.Actions) {
			return errorf("Number of actions in status actions and spec must match")
		}
		if err := actionSetStatus(as.Status); err != nil {
			return err
		}
	}
	return nil
}

func actionSetSpec(as *crv1alpha1.ActionSetSpec) error {
	if as == nil {
		return errorf("Spec must be non-nil")
	}
	for _, a := range as.Actions {
		if err := actionSpec(a); err != nil {
			return err
		}
	}
	return nil
}

func actionSpec(s crv1alpha1.ActionSpec) error {
	switch strings.ToLower(s.Object.Kind) {
	case param.StatefulSetKind:
		fallthrough
	case param.DeploymentKind:
		fallthrough
	case param.PVCKind:
		fallthrough
	case param.NamespaceKind:
		// Known types
	default:
		// Not a known type. ActionSet must specify API group and resource
		// name in order to populate `Object` TemplateParam
		if s.Object.APIVersion == "" || s.Object.Resource == "" {
			return errorf("Not a known object Kind %s. Action %s must specify Resource name and API version", s.Object.Kind, s.Name)
		}
	}
	return nil
}

func actionSetStatus(as *crv1alpha1.ActionSetStatus) error {
	if as == nil {
		return nil
	}
	if err := actionSetStatusActions(as.Actions); err != nil {
		return err
	}
	saw := map[crv1alpha1.State]bool{
		crv1alpha1.StatePending:  false,
		crv1alpha1.StateRunning:  false,
		crv1alpha1.StateFailed:   false,
		crv1alpha1.StateComplete: false,
	}
	for _, a := range as.Actions {
		for _, p := range a.Phases {
			if _, ok := saw[p.State]; !ok {
				return errorf("Action has unknown state '%s'", p.State)
			}
			for s := range saw {
				saw[s] = saw[s] || (p.State == s)
			}
		}
	}
	if _, ok := saw[as.State]; !ok {
		return errorf("ActionSet has unknown state '%s'", as.State)
	}
	if saw[crv1alpha1.StateRunning] || saw[crv1alpha1.StatePending] {
		if as.State == crv1alpha1.StateComplete {
			return errorf("ActionSet cannot be complete if any actions are not complete")
		}
	}
	return nil
}

func actionSetStatusActions(as []crv1alpha1.ActionStatus) error {
	for _, a := range as {
		var sawNotComplete bool
		var lastNonComplete crv1alpha1.State
		for _, p := range a.Phases {
			if sawNotComplete && p.State != crv1alpha1.StatePending {
				return errorf("Phases after a %s one must be pending", lastNonComplete)
			}
			if !sawNotComplete {
				lastNonComplete = p.State
			}
			sawNotComplete = p.State != crv1alpha1.StateComplete
		}
	}
	return nil
}

// Blueprint function validates the Blueprint and returns an error if it is invalid.
func Blueprint(bp *crv1alpha1.Blueprint) error {
	// TODO: Add blueprint validation.
	return nil
}

func ProfileSchema(p *crv1alpha1.Profile) error {
	if p.Location.Type != crv1alpha1.LocationTypeS3Compliant {
		return errorf("unknown or unsupported location type '%s'", p.Location.Type)
	}
	if p.Credential.Type != crv1alpha1.CredentialTypeKeyPair {
		return errorf("unknown or unsupported credential type '%s'", p.Credential.Type)
	}
	if p.Location.Bucket == "" {
		return errorf("S3 bucket not specified")
	}
	if p.Location.Endpoint == "" && p.Location.Region == "" {
		return errorf("S3 bucket region not specified")
	}
	if p.Credential.KeyPair.Secret.Name == "" {
		return errorf("secret for bucket credentials not specified")
	}
	if p.Credential.KeyPair.SecretField == "" || p.Credential.KeyPair.IDField == "" {
		return errorf("secret field or id field empty")
	}
	return nil
}

func ProfileBucket(ctx context.Context, p *crv1alpha1.Profile) error {
	bucketName := p.Location.Bucket
	givenRegion := p.Location.Region
	if givenRegion != "" {
		actualRegion, err := objectstore.GetS3BucketRegion(ctx, bucketName, givenRegion)
		if err != nil {
			return err
		}
		if actualRegion != givenRegion {
			return errorf("Incorrect region for bucket. Expected '%s', Got '%s'", actualRegion, givenRegion)
		}
	}
	return nil
}

func ReadAccess(ctx context.Context, p *crv1alpha1.Profile, cli kubernetes.Interface) error {
	secret := &objectstore.Secret{
		Type: objectstore.SecretTypeAwsAccessKey,
		Aws:  &objectstore.SecretAws{},
	}
	err := fillKVAwsCredentials(ctx, secret, p, cli)
	if err != nil {
		return err
	}
	pc := objectstore.ProviderConfig{
		Type:          objectstore.ProviderTypeS3,
		Endpoint:      p.Location.Endpoint,
		SkipSSLVerify: p.SkipSSLVerify,
	}
	provider, err := objectstore.NewProvider(ctx, pc, secret)
	if err != nil {
		return err
	}
	bucket, err := provider.GetBucket(ctx, p.Location.Bucket)
	if err != nil {
		return err
	}
	if _, err := bucket.ListDirectories(ctx); err != nil {
		return errorf("failed to list directories in bucket '%s'", p.Location.Bucket)
	}
	return nil
}

func WriteAccess(ctx context.Context, p *crv1alpha1.Profile, cli kubernetes.Interface) error {
	const objName = "sample"

	secret := &objectstore.Secret{
		Type: objectstore.SecretTypeAwsAccessKey,
		Aws:  &objectstore.SecretAws{},
	}
	err := fillKVAwsCredentials(ctx, secret, p, cli)
	if err != nil {
		return err
	}
	pc := objectstore.ProviderConfig{
		Type:          objectstore.ProviderTypeS3,
		Endpoint:      p.Location.Endpoint,
		SkipSSLVerify: p.SkipSSLVerify,
	}
	provider, err := objectstore.NewProvider(ctx, pc, secret)
	if err != nil {
		return err
	}
	bucket, err := provider.GetBucket(ctx, p.Location.Bucket)
	if err != nil {
		return err
	}
	data := []byte("sample content")
	if err := bucket.PutBytes(ctx, objName, data, nil); err != nil {
		return errorf("failed to write contents to bucket '%s'", p.Location.Bucket)
	}
	if err := bucket.Delete(ctx, objName); err != nil {
		return errorf("failed to delete contents in bucket '%s'", p.Location.Bucket)
	}
	return nil
}

func fillKVAwsCredentials(ctx context.Context, ss *objectstore.Secret, p *crv1alpha1.Profile, cli kubernetes.Interface) error {
	kp := p.Credential.KeyPair
	if kp == nil {
		return errorf("invalid credentials kv cannot be nil")
	}
	s, err := cli.CoreV1().Secrets(kp.Secret.Namespace).Get(kp.Secret.Name, metav1.GetOptions{})
	if err != nil {
		return errorf("could not fetch the secret specified in credential")
	}
	if _, ok := s.Data[kp.IDField]; !ok {
		return errorf("Key '%s' not found in secret '%s:%s'", kp.IDField, s.GetNamespace(), s.GetName())
	}
	if _, ok := s.Data[kp.SecretField]; !ok {
		return errorf("Value '%s' not found in secret '%s:%s'", kp.SecretField, s.GetNamespace(), s.GetName())
	}
	ss.Aws.AccessKeyID = string(s.Data[kp.IDField])
	ss.Aws.SecretAccessKey = string(s.Data[kp.SecretField])
	return nil
}
