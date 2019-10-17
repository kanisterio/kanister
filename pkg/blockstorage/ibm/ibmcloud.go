// Copyright 2019 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ibm

// IBM Cloud Block storage

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	ibmprov "github.com/IBM/ibmcloud-storage-volume-lib/lib/provider"
	"github.com/jpillora/backoff"
	"github.com/kanisterio/kanister/pkg/poll"
	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg/blockstorage"
	bsibmutils "github.com/kanisterio/kanister/pkg/blockstorage/ibm/utils"
	ktags "github.com/kanisterio/kanister/pkg/blockstorage/tags"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
)

var _ blockstorage.Provider = (*ibmCloud)(nil)

type ibmCloud struct {
	cli *client
}

const (
	maxRetries     = 10
	defaultTimeout = time.Duration(time.Minute * 5)
)

func (s *ibmCloud) Type() blockstorage.Type {
	if s.cli.SLCfg.SoftlayerFileEnabled {
		return blockstorage.TypeSoftlayerFile
	}
	return blockstorage.TypeSoftlayerBlock
}

// NewProvider returns a provider for the IBM Cloud
func NewProvider(ctx context.Context, args map[string]string) (blockstorage.Provider, error) {
	ibmCli, err := newClient(ctx, args)
	return &ibmCloud{cli: ibmCli}, err
}

func (s *ibmCloud) VolumeCreate(ctx context.Context, volume blockstorage.Volume) (*blockstorage.Volume, error) {
	newVol := ibmprov.Volume{}
	newVol.VolumeType = ibmprov.VolumeType(volume.VolumeType)
	newVol.ProviderType = ibmprov.VolumeProviderType(volume.Attributes[ProviderTypeAttName])
	if volume.Iops > 0 {
		iops := strconv.Itoa(int(volume.Iops))
		newVol.Iops = &iops
	}
	if tier, ok := volume.Attributes[TierAttName]; ok {
		newVol.Tier = &tier
	}
	size := int(volume.Size)
	newVol.Capacity = &size
	newVol.VolumeNotes = blockstorage.KeyValueToMap(volume.Tags)
	newVol.Az = s.cli.SLCfg.SoftlayerDataCenter
	log.Debug().Print("Creating new volume", field.M{"volume": newVol})
	volR, err := s.cli.Service.CreateVolume(newVol)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to create volume. %s", err.Error()))
	}
	log.Debug().Print("New volume created", field.M{"volume": volR})
	return s.volumeParse(ctx, volR)
}

func (s *ibmCloud) VolumeGet(ctx context.Context, id string, zone string) (*blockstorage.Volume, error) {
	vol, err := s.cli.Service.GetVolume(id)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get volume with id %s", id)
	}
	log.Debug().Print("Got volume from cloud provider", field.M{"Volume": vol})
	return s.volumeParse(ctx, vol)
}

func (s *ibmCloud) volumeParse(ctx context.Context, vol *ibmprov.Volume) (*blockstorage.Volume, error) {
	tags := []*blockstorage.KeyValue(nil)
	for k, v := range vol.VolumeNotes {
		tags = append(tags, &blockstorage.KeyValue{Key: k, Value: v})
	}
	var iops int64
	if vol.Iops != nil {
		iops, _ = strconv.ParseInt(*vol.Iops, 10, 64)
	}

	attribs := map[string]string{
		ProviderAttName:      string(vol.Provider),
		ProviderTypeAttName:  string(vol.ProviderType),
		SnapshotSpaceAttName: "",
		TierAttName:          "",
		BillingTypeAttName:   vol.BillingType,
		RegionAttName:        vol.Region,
	}

	if vol.Tier != nil {
		attribs[TierAttName] = *vol.Tier
	}

	if vol.SnapshotSpace != nil {
		attribs[SnapshotSpaceAttName] = strconv.Itoa(*vol.SnapshotSpace)
	}

	if vol.LunID == "" && string(vol.Provider) == s.cli.SLCfg.SoftlayerBlockProviderName {
		return nil, errors.New("LunID is missing from Volume info")
	}
	attribs[LunIDAttName] = vol.LunID

	if len(vol.IscsiTargetIPAddresses) < 0 {
		return nil, errors.New("IscsiTargetIPAddresses are missing from Volume info")
	}
	if len(vol.Attributes) > 0 {
		for k, v := range vol.Attributes {
			attribs[k] = v
		}
	}

	attribs[TargetIPsAttName] = strings.Join(vol.IscsiTargetIPAddresses, ",")

	return &blockstorage.Volume{
		Type:         s.Type(),
		ID:           vol.VolumeID,
		Az:           s.cli.SLCfg.SoftlayerDataCenter, // Due to the bug in IBM lib vol.Az,
		Encrypted:    false,
		VolumeType:   string(vol.VolumeType),
		Size:         int64(*vol.Capacity),
		Tags:         tags,
		Iops:         iops,
		CreationTime: blockstorage.TimeStamp(vol.CreationTime),
		Attributes:   attribs,
	}, nil
}

func (s *ibmCloud) VolumesList(ctx context.Context, tags map[string]string, zone string) ([]*blockstorage.Volume, error) {
	var vols []*blockstorage.Volume
	ibmVols, err := s.cli.Service.ListVolumes(tags)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to list volumes with tags %v", tags)
	}

	for _, v := range ibmVols {
		pvol, err := s.volumeParse(ctx, v)
		if err != nil {
			return nil, errors.Errorf("Failed to parse vol %s", err.Error())
		}
		vols = append(vols, pvol)
	}
	return vols, nil
}

func (s *ibmCloud) snapshotParse(ctx context.Context, snap *ibmprov.Snapshot) *blockstorage.Snapshot {
	tags := []*blockstorage.KeyValue(nil)
	for k, v := range snap.SnapshotTags {
		tags = append(tags, &blockstorage.KeyValue{Key: k, Value: v})
	}

	vol := &blockstorage.Volume{
		Type: s.Type(),
		ID:   snap.VolumeID,
	}

	snapSize := int64(0)
	if snap.SnapshotSize != nil {
		snapSize = int64(*snap.SnapshotSize)
	}

	return &blockstorage.Snapshot{
		ID:           snap.SnapshotID,
		Tags:         tags,
		Type:         s.Type(),
		Encrypted:    false,
		Size:         snapSize,
		Region:       snap.Region,
		Volume:       vol,
		CreationTime: blockstorage.TimeStamp(snap.SnapshotCreationTime),
	}
}

func (s *ibmCloud) SnapshotsList(ctx context.Context, tags map[string]string) ([]*blockstorage.Snapshot, error) {
	var snaps []*blockstorage.Snapshot
	// IBM doens't support tag for snapshot list.
	// Getting all of them
	ibmsnaps, err := s.cli.Service.ListSnapshots()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to list Snapshots")
	}

	for _, snap := range ibmsnaps {
		snaps = append(snaps, s.snapshotParse(ctx, snap))
	}
	return snaps, nil
}

// SnapshotCopy copies snapshot 'from' to 'to'. Follows aws restrictions regarding encryption;
// i.e., copying unencrypted to encrypted snapshot is allowed but not vice versa.
func (s *ibmCloud) SnapshotCopy(ctx context.Context, from, to blockstorage.Snapshot) (*blockstorage.Snapshot, error) {
	return nil, errors.New("Not implemented")
}

func (s *ibmCloud) SnapshotCreate(ctx context.Context, volume blockstorage.Volume, tags map[string]string) (*blockstorage.Snapshot, error) {
	alltags := ktags.GetTags(tags)
	ibmvol, err := s.cli.Service.GetVolume(volume.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get Volume for Snapshot creation, volume_id :%s", volume.ID)
	}

	if ibmvol.SnapshotSpace == nil {
		log.Debug().Print("Ordering snapshot space for volume", field.M{"Volume": ibmvol})
		ibmvol.SnapshotSpace = ibmvol.Capacity
		err = s.cli.Service.OrderSnapshot(*ibmvol)
		if err != nil {
			if strings.Contains(err.Error(), "already has snapshot space") != true {
				return nil, errors.Wrapf(err, "Failed to order Snapshot space, volume_id :%s", volume.ID)
			}
		}
		wctx, wcancel := context.WithTimeout(ctx, defaultTimeout)
		defer wcancel()
		err = waitforSnapSpaceOrder(wctx, s.cli, volume.ID)
		if err != nil {
			return nil, errors.Wrapf(err, "Wait is expired for order Snapshot space, volume_id :%s", volume.ID)
		}
	}
	log.Debug().Print("Creating snapshot", field.M{"Volume": ibmvol})
	snap, err := s.cli.Service.CreateSnapshot(ibmvol, alltags)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create snapshot, volume_id: %s", volume.ID)
	}
	log.Debug().Print("New snapshot was created ", field.M{"Snapshot": snap})

	return s.snapshotParse(ctx, snap), nil
}

func (s *ibmCloud) SnapshotCreateWaitForCompletion(ctx context.Context, snap *blockstorage.Snapshot) error {
	return nil
}

func (s *ibmCloud) SnapshotDelete(ctx context.Context, snapshot *blockstorage.Snapshot) error {
	snap, err := s.cli.Service.GetSnapshot(snapshot.ID)
	if err != nil {
		return errors.Wrapf(err, "Failed to get Snapshot for deletion, snapshot_id %s", snapshot.ID)
	}
	return s.cli.Service.DeleteSnapshot(snap)
}

func (s *ibmCloud) SnapshotGet(ctx context.Context, id string) (*blockstorage.Snapshot, error) {
	snap, err := s.cli.Service.GetSnapshot(id)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get Snapshot, snapshot_id %s", id)
	}
	return s.snapshotParse(ctx, snap), nil
}

func (s *ibmCloud) VolumeDelete(ctx context.Context, volume *blockstorage.Volume) error {
	ibmvol, err := s.cli.Service.GetVolume(volume.ID)
	if err != nil {
		return errors.Wrapf(err, "Failed to get Volume for deletion, volume_id :%s", volume.ID)
	}
	return s.cli.Service.DeleteVolume(ibmvol)
}

func (s *ibmCloud) SetTags(ctx context.Context, resource interface{}, tags map[string]string) error {
	log.Print("IBM storage lib does not support SetTags")
	return nil
}

func (s *ibmCloud) VolumeCreateFromSnapshot(ctx context.Context, snapshot blockstorage.Snapshot, tags map[string]string) (*blockstorage.Volume, error) {
	if snapshot.Volume == nil {
		return nil, errors.New("Snapshot volume information not available")
	}

	// Incorporate pre-existing tags.
	for _, tag := range snapshot.Volume.Tags {
		if _, found := tags[tag.Key]; !found {
			tags[tag.Key] = tag.Value
		}
	}
	snap, err := s.cli.Service.GetSnapshot(snapshot.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get Snapshot for Volume Restore, snapshot_id %s", snapshot.ID)
	}
	// Temporary fix for Softlayer backend bug
	// GetSnapshot doens't return correct Volume info
	snap.VolumeID = snapshot.Volume.ID
	snap.Volume.VolumeID = snapshot.Volume.ID
	log.Debug().Print("Creating Volume from Snapshot with new volume ID", field.M{"inputSnapshot": snap})

	vol, err := s.cli.Service.CreateVolumeFromSnapshot(*snap, tags)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Volume from Snapshot, snapshot_id %s", snapshot.ID)
	}

	if err = bsibmutils.AuthorizeSoftLayerFileHosts(ctx, vol, s.cli.Service); err != nil {
		return nil, errors.Wrap(err, "Failed to add Authorized Hosts to new volume, without Authorized Hosts ibm storage will not be able to mount volume to kubernetes node")
	}

	return s.VolumeGet(ctx, vol.VolumeID, snapshot.Volume.Az)
}

func waitforSnapSpaceOrder(ctx context.Context, cli *client, id string) error {
	snapWaitBackoff := backoff.Backoff{
		Factor: 2,
		Jitter: false,
		Min:    1 * time.Second,
		Max:    10 * time.Second,
	}
	return poll.WaitWithBackoff(ctx, snapWaitBackoff, func(ctx context.Context) (bool, error) {
		vol, err := cli.Service.GetVolume(id)
		if err != nil {
			return false, errors.Wrapf(err, "Failed to get volume, volume_id: %s", id)
		}

		if vol.SnapshotSpace != nil {
			log.Debug().Print("Volume has snapshot space now", field.M{"volume_id": id})
			return true, nil
		}
		log.Debug().Print("Still waiting for Snapshor Space order", field.M{"volume_id": id})
		return false, nil
	})

}
