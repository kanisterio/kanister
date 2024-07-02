//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright 2024 The Kanister Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// Code generated by deepcopy-gen. DO NOT EDIT.

package v1alpha1

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ActionProgress) DeepCopyInto(out *ActionProgress) {
	*out = *in
	if in.LastTransitionTime != nil {
		in, out := &in.LastTransitionTime, &out.LastTransitionTime
		*out = (*in).DeepCopy()
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ActionProgress.
func (in *ActionProgress) DeepCopy() *ActionProgress {
	if in == nil {
		return nil
	}
	out := new(ActionProgress)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ActionSet) DeepCopyInto(out *ActionSet) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	if in.Spec != nil {
		in, out := &in.Spec, &out.Spec
		*out = new(ActionSetSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Status != nil {
		in, out := &in.Status, &out.Status
		*out = new(ActionSetStatus)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ActionSet.
func (in *ActionSet) DeepCopy() *ActionSet {
	if in == nil {
		return nil
	}
	out := new(ActionSet)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ActionSet) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ActionSetList) DeepCopyInto(out *ActionSetList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]*ActionSet, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(ActionSet)
				(*in).DeepCopyInto(*out)
			}
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ActionSetList.
func (in *ActionSetList) DeepCopy() *ActionSetList {
	if in == nil {
		return nil
	}
	out := new(ActionSetList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ActionSetList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ActionSetSpec) DeepCopyInto(out *ActionSetSpec) {
	*out = *in
	if in.Actions != nil {
		in, out := &in.Actions, &out.Actions
		*out = make([]ActionSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ActionSetSpec.
func (in *ActionSetSpec) DeepCopy() *ActionSetSpec {
	if in == nil {
		return nil
	}
	out := new(ActionSetSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ActionSetStatus) DeepCopyInto(out *ActionSetStatus) {
	*out = *in
	if in.Actions != nil {
		in, out := &in.Actions, &out.Actions
		*out = make([]ActionStatus, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	out.Error = in.Error
	in.Progress.DeepCopyInto(&out.Progress)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ActionSetStatus.
func (in *ActionSetStatus) DeepCopy() *ActionSetStatus {
	if in == nil {
		return nil
	}
	out := new(ActionSetStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ActionSpec) DeepCopyInto(out *ActionSpec) {
	*out = *in
	out.Object = in.Object
	if in.Artifacts != nil {
		in, out := &in.Artifacts, &out.Artifacts
		*out = make(map[string]Artifact, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.ConfigMaps != nil {
		in, out := &in.ConfigMaps, &out.ConfigMaps
		*out = make(map[string]ObjectReference, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Secrets != nil {
		in, out := &in.Secrets, &out.Secrets
		*out = make(map[string]ObjectReference, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Profile != nil {
		in, out := &in.Profile, &out.Profile
		*out = new(ObjectReference)
		**out = **in
	}
	if in.RepositoryServer != nil {
		in, out := &in.RepositoryServer, &out.RepositoryServer
		*out = new(ObjectReference)
		**out = **in
	}
	out.PodOverride = in.PodOverride.DeepCopy()
	if in.Options != nil {
		in, out := &in.Options, &out.Options
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ActionSpec.
func (in *ActionSpec) DeepCopy() *ActionSpec {
	if in == nil {
		return nil
	}
	out := new(ActionSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ActionStatus) DeepCopyInto(out *ActionStatus) {
	*out = *in
	out.Object = in.Object
	if in.Phases != nil {
		in, out := &in.Phases, &out.Phases
		*out = make([]Phase, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Artifacts != nil {
		in, out := &in.Artifacts, &out.Artifacts
		*out = make(map[string]Artifact, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	in.DeferPhase.DeepCopyInto(&out.DeferPhase)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ActionStatus.
func (in *ActionStatus) DeepCopy() *ActionStatus {
	if in == nil {
		return nil
	}
	out := new(ActionStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Artifact) DeepCopyInto(out *Artifact) {
	*out = *in
	if in.KeyValue != nil {
		in, out := &in.KeyValue, &out.KeyValue
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Artifact.
func (in *Artifact) DeepCopy() *Artifact {
	if in == nil {
		return nil
	}
	out := new(Artifact)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Blueprint) DeepCopyInto(out *Blueprint) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	if in.Actions != nil {
		in, out := &in.Actions, &out.Actions
		*out = make(map[string]*BlueprintAction, len(*in))
		for key, val := range *in {
			var outVal *BlueprintAction
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = new(BlueprintAction)
				(*in).DeepCopyInto(*out)
			}
			(*out)[key] = outVal
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Blueprint.
func (in *Blueprint) DeepCopy() *Blueprint {
	if in == nil {
		return nil
	}
	out := new(Blueprint)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Blueprint) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BlueprintAction) DeepCopyInto(out *BlueprintAction) {
	*out = *in
	if in.ConfigMapNames != nil {
		in, out := &in.ConfigMapNames, &out.ConfigMapNames
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.SecretNames != nil {
		in, out := &in.SecretNames, &out.SecretNames
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.InputArtifactNames != nil {
		in, out := &in.InputArtifactNames, &out.InputArtifactNames
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.OutputArtifacts != nil {
		in, out := &in.OutputArtifacts, &out.OutputArtifacts
		*out = make(map[string]Artifact, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.Phases != nil {
		in, out := &in.Phases, &out.Phases
		*out = make([]BlueprintPhase, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.DeferPhase != nil {
		in, out := &in.DeferPhase, &out.DeferPhase
		*out = (*in).DeepCopy()
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BlueprintAction.
func (in *BlueprintAction) DeepCopy() *BlueprintAction {
	if in == nil {
		return nil
	}
	out := new(BlueprintAction)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BlueprintList) DeepCopyInto(out *BlueprintList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]*Blueprint, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(Blueprint)
				(*in).DeepCopyInto(*out)
			}
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BlueprintList.
func (in *BlueprintList) DeepCopy() *BlueprintList {
	if in == nil {
		return nil
	}
	out := new(BlueprintList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *BlueprintList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BlueprintPhase.
func (in *BlueprintPhase) DeepCopy() *BlueprintPhase {
	if in == nil {
		return nil
	}
	out := new(BlueprintPhase)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CacheSizeSettings) DeepCopyInto(out *CacheSizeSettings) {
	*out = *in
	if in.Metadata != nil {
		in, out := &in.Metadata, &out.Metadata
		*out = new(int)
		**out = **in
	}
	if in.Content != nil {
		in, out := &in.Content, &out.Content
		*out = new(int)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CacheSizeSettings.
func (in *CacheSizeSettings) DeepCopy() *CacheSizeSettings {
	if in == nil {
		return nil
	}
	out := new(CacheSizeSettings)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Configuration) DeepCopyInto(out *Configuration) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Configuration.
func (in *Configuration) DeepCopy() *Configuration {
	if in == nil {
		return nil
	}
	out := new(Configuration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Credential) DeepCopyInto(out *Credential) {
	*out = *in
	if in.KeyPair != nil {
		in, out := &in.KeyPair, &out.KeyPair
		*out = new(KeyPair)
		**out = **in
	}
	if in.Secret != nil {
		in, out := &in.Secret, &out.Secret
		*out = new(ObjectReference)
		**out = **in
	}
	if in.KopiaServerSecret != nil {
		in, out := &in.KopiaServerSecret, &out.KopiaServerSecret
		*out = new(KopiaServerSecret)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Credential.
func (in *Credential) DeepCopy() *Credential {
	if in == nil {
		return nil
	}
	out := new(Credential)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Error) DeepCopyInto(out *Error) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Error.
func (in *Error) DeepCopy() *Error {
	if in == nil {
		return nil
	}
	out := new(Error)
	in.DeepCopyInto(out)
	return out
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JSONMap.
func (in JSONMap) DeepCopy() JSONMap {
	if in == nil {
		return nil
	}
	out := new(JSONMap)
	in.DeepCopyInto(out)
	return *out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KeyPair) DeepCopyInto(out *KeyPair) {
	*out = *in
	out.Secret = in.Secret
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KeyPair.
func (in *KeyPair) DeepCopy() *KeyPair {
	if in == nil {
		return nil
	}
	out := new(KeyPair)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KopiaServerSecret) DeepCopyInto(out *KopiaServerSecret) {
	*out = *in
	if in.UserPassphrase != nil {
		in, out := &in.UserPassphrase, &out.UserPassphrase
		*out = new(KopiaServerSecretRef)
		(*in).DeepCopyInto(*out)
	}
	if in.TLSCert != nil {
		in, out := &in.TLSCert, &out.TLSCert
		*out = new(KopiaServerSecretRef)
		(*in).DeepCopyInto(*out)
	}
	if in.ConnectOptions != nil {
		in, out := &in.ConnectOptions, &out.ConnectOptions
		*out = make(map[string]int, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KopiaServerSecret.
func (in *KopiaServerSecret) DeepCopy() *KopiaServerSecret {
	if in == nil {
		return nil
	}
	out := new(KopiaServerSecret)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KopiaServerSecretRef) DeepCopyInto(out *KopiaServerSecretRef) {
	*out = *in
	if in.Secret != nil {
		in, out := &in.Secret, &out.Secret
		*out = new(ObjectReference)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KopiaServerSecretRef.
func (in *KopiaServerSecretRef) DeepCopy() *KopiaServerSecretRef {
	if in == nil {
		return nil
	}
	out := new(KopiaServerSecretRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Location) DeepCopyInto(out *Location) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Location.
func (in *Location) DeepCopy() *Location {
	if in == nil {
		return nil
	}
	out := new(Location)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ObjectReference) DeepCopyInto(out *ObjectReference) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ObjectReference.
func (in *ObjectReference) DeepCopy() *ObjectReference {
	if in == nil {
		return nil
	}
	out := new(ObjectReference)
	in.DeepCopyInto(out)
	return out
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Phase.
func (in *Phase) DeepCopy() *Phase {
	if in == nil {
		return nil
	}
	out := new(Phase)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PhaseProgress) DeepCopyInto(out *PhaseProgress) {
	*out = *in
	if in.LastTransitionTime != nil {
		in, out := &in.LastTransitionTime, &out.LastTransitionTime
		*out = (*in).DeepCopy()
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PhaseProgress.
func (in *PhaseProgress) DeepCopy() *PhaseProgress {
	if in == nil {
		return nil
	}
	out := new(PhaseProgress)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Profile) DeepCopyInto(out *Profile) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Location = in.Location
	in.Credential.DeepCopyInto(&out.Credential)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Profile.
func (in *Profile) DeepCopy() *Profile {
	if in == nil {
		return nil
	}
	out := new(Profile)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Profile) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ProfileList) DeepCopyInto(out *ProfileList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]*Profile, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(Profile)
				(*in).DeepCopyInto(*out)
			}
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProfileList.
func (in *ProfileList) DeepCopy() *ProfileList {
	if in == nil {
		return nil
	}
	out := new(ProfileList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ProfileList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Repository) DeepCopyInto(out *Repository) {
	*out = *in
	out.PasswordSecretRef = in.PasswordSecretRef
	in.CacheSizeSettings.DeepCopyInto(&out.CacheSizeSettings)
	out.Configuration = in.Configuration
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Repository.
func (in *Repository) DeepCopy() *Repository {
	if in == nil {
		return nil
	}
	out := new(Repository)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RepositoryServer) DeepCopyInto(out *RepositoryServer) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RepositoryServer.
func (in *RepositoryServer) DeepCopy() *RepositoryServer {
	if in == nil {
		return nil
	}
	out := new(RepositoryServer)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RepositoryServer) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RepositoryServerList) DeepCopyInto(out *RepositoryServerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]RepositoryServer, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RepositoryServerList.
func (in *RepositoryServerList) DeepCopy() *RepositoryServerList {
	if in == nil {
		return nil
	}
	out := new(RepositoryServerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RepositoryServerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RepositoryServerSpec) DeepCopyInto(out *RepositoryServerSpec) {
	*out = *in
	out.Storage = in.Storage
	in.Repository.DeepCopyInto(&out.Repository)
	out.Server = in.Server
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RepositoryServerSpec.
func (in *RepositoryServerSpec) DeepCopy() *RepositoryServerSpec {
	if in == nil {
		return nil
	}
	out := new(RepositoryServerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RepositoryServerStatus) DeepCopyInto(out *RepositoryServerStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	out.ServerInfo = in.ServerInfo
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RepositoryServerStatus.
func (in *RepositoryServerStatus) DeepCopy() *RepositoryServerStatus {
	if in == nil {
		return nil
	}
	out := new(RepositoryServerStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Server) DeepCopyInto(out *Server) {
	*out = *in
	out.UserAccess = in.UserAccess
	out.AdminSecretRef = in.AdminSecretRef
	out.TLSSecretRef = in.TLSSecretRef
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Server.
func (in *Server) DeepCopy() *Server {
	if in == nil {
		return nil
	}
	out := new(Server)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServerInfo) DeepCopyInto(out *ServerInfo) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServerInfo.
func (in *ServerInfo) DeepCopy() *ServerInfo {
	if in == nil {
		return nil
	}
	out := new(ServerInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Storage) DeepCopyInto(out *Storage) {
	*out = *in
	out.SecretRef = in.SecretRef
	out.CredentialSecretRef = in.CredentialSecretRef
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Storage.
func (in *Storage) DeepCopy() *Storage {
	if in == nil {
		return nil
	}
	out := new(Storage)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UserAccess) DeepCopyInto(out *UserAccess) {
	*out = *in
	out.UserAccessSecretRef = in.UserAccessSecretRef
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UserAccess.
func (in *UserAccess) DeepCopy() *UserAccess {
	if in == nil {
		return nil
	}
	out := new(UserAccess)
	in.DeepCopyInto(out)
	return out
}
