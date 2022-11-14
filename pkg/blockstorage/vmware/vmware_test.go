package vmware

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/pkg/errors"
	govmomitask "github.com/vmware/govmomi/task"
	vapitags "github.com/vmware/govmomi/vapi/tags"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
	"github.com/vmware/govmomi/vim25/xml"
	vslmtypes "github.com/vmware/govmomi/vslm/types"
	. "gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/blockstorage"
)

func Test(t *testing.T) { TestingT(t) }

type VMWareSuite struct{}

var _ = Suite(&VMWareSuite{})

func (s *VMWareSuite) TestURLParse(c *C) {
	for _, tc := range []struct {
		config       map[string]string
		errCheck     Checker
		expErrString string
	}{
		{
			config:       map[string]string{},
			errCheck:     NotNil,
			expErrString: "Failed to find VSphere endpoint value",
		},
		{
			config: map[string]string{
				VSphereEndpointKey: "ep",
			},
			errCheck:     NotNil,
			expErrString: "Failed to find VSphere username value",
		},
		{
			config: map[string]string{
				VSphereEndpointKey: "ep",
				VSphereUsernameKey: "user",
			},
			errCheck:     NotNil,
			expErrString: "Failed to find VSphere password value",
		},
		{ // until we can run against a VIM setup this will always fail.
			config: map[string]string{
				VSphereEndpointKey: "ep",
				VSphereUsernameKey: "user",
				VSpherePasswordKey: "pass",
			},
			errCheck:     NotNil,
			expErrString: "Failed to create VIM client",
		},
	} {
		_, err := NewProvider(tc.config)
		c.Check(err, tc.errCheck)
		if err != nil {
			c.Assert(err, ErrorMatches, ".*"+tc.expErrString+".*")
		}
	}
}

func (s *VMWareSuite) TestIsParaVirtualized(c *C) {
	// the constructor needs VIM so just check the parsing of the config map.

	config := map[string]string{}
	c.Assert(false, Equals, configIsParaVirtualized(config))
	config[VSphereIsParaVirtualizedKey] = "false"
	c.Assert(false, Equals, configIsParaVirtualized(config))
	config[VSphereIsParaVirtualizedKey] = "true"
	c.Assert(true, Equals, configIsParaVirtualized(config))
	config[VSphereIsParaVirtualizedKey] = "TRUE"
	c.Assert(true, Equals, configIsParaVirtualized(config))
	config[VSphereIsParaVirtualizedKey] = "1"
	c.Assert(true, Equals, configIsParaVirtualized(config))

	fcd := &FcdProvider{}
	c.Assert(false, Equals, fcd.IsParaVirtualized())
	fcd.isParaVirtualized = true
	c.Assert(true, Equals, fcd.IsParaVirtualized())

	// failed operations
	v, err := fcd.VolumeCreateFromSnapshot(context.Background(), blockstorage.Snapshot{}, nil)
	c.Assert(true, Equals, errors.Is(err, ErrNotSupportedWithParaVirtualizedVolumes))
	c.Assert(v, IsNil)
}

func (s *VMWareSuite) TestTimeoutEnvSetting(c *C) {
	tempEnv := os.Getenv(vmWareTimeoutMinEnv)
	os.Unsetenv(vmWareTimeoutMinEnv)
	timeout := time.Duration(getEnvAsIntOrDefault(vmWareTimeoutMinEnv, int(defaultWaitTime/time.Minute))) * time.Minute
	c.Assert(timeout, Equals, defaultWaitTime)

	os.Setenv(vmWareTimeoutMinEnv, "7")
	timeout = time.Duration(getEnvAsIntOrDefault(vmWareTimeoutMinEnv, int(defaultWaitTime/time.Minute))) * time.Minute
	c.Assert(timeout, Equals, 7*time.Minute)

	os.Setenv(vmWareTimeoutMinEnv, "badValue")
	timeout = time.Duration(getEnvAsIntOrDefault(vmWareTimeoutMinEnv, int(defaultWaitTime/time.Minute))) * time.Minute
	c.Assert(timeout, Equals, defaultWaitTime)

	os.Setenv(vmWareTimeoutMinEnv, "-1")
	timeout = time.Duration(getEnvAsIntOrDefault(vmWareTimeoutMinEnv, int(defaultWaitTime/time.Minute))) * time.Minute
	c.Assert(timeout, Equals, defaultWaitTime)

	os.Setenv(vmWareTimeoutMinEnv, "0")
	timeout = time.Duration(getEnvAsIntOrDefault(vmWareTimeoutMinEnv, int(defaultWaitTime/time.Minute))) * time.Minute
	c.Assert(timeout, Equals, defaultWaitTime)

	timeout = time.Duration(getEnvAsIntOrDefault("someotherenv", 5)) * time.Minute
	c.Assert(timeout, Equals, 5*time.Minute)

	os.Setenv(vmWareTimeoutMinEnv, tempEnv)
}

func (s *VMWareSuite) TestGetSnapshotIDsFromTags(c *C) {
	for _, tc := range []struct {
		catTags    []vapitags.Tag
		tags       map[string]string
		errChecker Checker
		snapIDs    []string
	}{
		{
			catTags: []vapitags.Tag{
				{Name: "v1:s1:k1:v1"},
				{Name: "v1:s1:k2:v2"},
				{Name: "v1:s2:k1:v1"},
			},
			tags: map[string]string{
				"k1": "v1",
				"k2": "v2",
			},
			snapIDs:    []string{"v1:s1"},
			errChecker: IsNil,
		},
		{
			catTags: []vapitags.Tag{
				{Name: "v1:s1:k1:v1"},
				{Name: "v1:s1:k2:v2"},
				{Name: "v1:s2:k1:v1"},
			},
			tags: map[string]string{
				"k1": "v1",
			},
			snapIDs:    []string{"v1:s1", "v1:s2"},
			errChecker: IsNil,
		},
		{
			catTags: []vapitags.Tag{
				{Name: "v1:s1:k1:v1"},
				{Name: "v1:s1:k2:v2"},
				{Name: "v1:s2:k1:v1"},
			},
			snapIDs:    []string{"v1:s1", "v1:s2"},
			errChecker: IsNil,
		},
		{
			catTags: []vapitags.Tag{
				{Name: "v1:s1k1:v1"},
			},
			tags: map[string]string{
				"k1": "v1",
			},
			errChecker: NotNil,
		},
	} {
		fp := &FcdProvider{}
		snapIDs, err := fp.getSnapshotIDsFromTags(tc.catTags, tc.tags)
		c.Assert(err, tc.errChecker)
		if tc.errChecker == IsNil {
			sort.Strings(snapIDs)
			sort.Strings(tc.snapIDs)
			c.Assert(snapIDs, DeepEquals, tc.snapIDs)
		}
	}
}

func (s *VMWareSuite) TestSetTagsSnapshot(c *C) {
	ctx := context.Background()
	for _, tc := range []struct {
		catID         string
		snapshot      *blockstorage.Snapshot
		tags          map[string]string
		errChecker    Checker
		expNumCreates int

		errCreateTag error
	}{
		{ // success
			catID:    "catid",
			snapshot: &blockstorage.Snapshot{ID: "volid:snapid"},
			tags: map[string]string{
				"t1": "v1",
				"t2": "v2",
			},
			expNumCreates: 2,
			errChecker:    IsNil,
		},
		{ // idempotent creates
			catID:    "catid",
			snapshot: &blockstorage.Snapshot{ID: "volid:snapid"},
			tags: map[string]string{
				"t1": "v1",
				"t2": "v2",
			},
			expNumCreates: 2,
			errCreateTag:  fmt.Errorf("ALREADY_EXISTS"),
			errChecker:    IsNil,
		},
		{ // create failure
			catID:    "catid",
			snapshot: &blockstorage.Snapshot{ID: "volid:snapid"},
			tags: map[string]string{
				"t1": "v1",
				"t2": "v2",
			},
			expNumCreates: 2,
			errCreateTag:  fmt.Errorf("bad create"),
			errChecker:    NotNil,
		},
		{ // malformed id
			catID:      "catid",
			snapshot:   &blockstorage.Snapshot{ID: "volidsnapid"},
			errChecker: NotNil,
		},
		{ // nil snapshot
			catID:      "catid",
			errChecker: NotNil,
		},
		{ // empty id, No error, not supported
			catID:      "",
			errChecker: IsNil,
		},
	} {
		ftm := &fakeTagManager{
			errCreateTag: tc.errCreateTag,
		}
		provider := &FcdProvider{
			categoryID: tc.catID,
			tagManager: ftm,
		}
		err := provider.setSnapshotTags(ctx, tc.snapshot, tc.tags)
		c.Assert(err, tc.errChecker)
		if tc.errChecker == IsNil {
			c.Assert(ftm.numCreates, Equals, tc.expNumCreates)
		}
	}
}

func (s *VMWareSuite) TestDeleteTagsSnapshot(c *C) {
	ctx := context.Background()
	for _, tc := range []struct {
		catID         string
		snapshot      *blockstorage.Snapshot
		errChecker    Checker
		expNumDeletes int

		retGetTagsForCategory []vapitags.Tag
		errGetTagsForCategory error
		errDeleteTag          error
	}{
		{ // success deleting tags
			catID:    "catid",
			snapshot: &blockstorage.Snapshot{ID: "volid:snapid"},
			retGetTagsForCategory: []vapitags.Tag{
				{Name: "volid:snapid:t1:v1"},
				{Name: "volid:snapid:t2:v2"},
				{Name: "volid:snapid2:t1:v1"},
			},
			expNumDeletes: 2,
			errChecker:    IsNil,
		},
		{ // error deleting tags
			catID:    "catid",
			snapshot: &blockstorage.Snapshot{ID: "volid:snapid"},
			retGetTagsForCategory: []vapitags.Tag{
				{Name: "volid:snapid:t1:v1"},
				{Name: "volid:snapid:t2:v2"},
			},
			errDeleteTag: fmt.Errorf("Failed to delete tag"),
			errChecker:   NotNil,
		},
		{ // error parsing tags
			catID:    "catid",
			snapshot: &blockstorage.Snapshot{ID: "volid:snapid"},
			retGetTagsForCategory: []vapitags.Tag{
				{Name: "volid:snapidt1v1"},
				{Name: "volid:snapid:t2:v2"},
			},
			errChecker: NotNil,
		},
		{ // error fetching tags
			catID:                 "catid",
			snapshot:              &blockstorage.Snapshot{ID: "volid:snapid"},
			errGetTagsForCategory: fmt.Errorf("Failed to get tags"),
			errChecker:            NotNil,
		},
		{ // malformed id
			catID:      "catid",
			snapshot:   &blockstorage.Snapshot{ID: "volidsnapid"},
			errChecker: NotNil,
		},
		{ // nil snapshot
			catID:      "catid",
			errChecker: NotNil,
		},
		{ // empty id, No error, not supported
			catID:      "",
			errChecker: IsNil,
		},
	} {
		ftm := &fakeTagManager{
			retGetTagsForCategory: tc.retGetTagsForCategory,
			errGetTagsForCategory: tc.errGetTagsForCategory,
			errDeleteTag:          tc.errDeleteTag,
		}
		provider := &FcdProvider{
			categoryID: tc.catID,
			tagManager: ftm,
		}
		err := provider.deleteSnapshotTags(ctx, tc.snapshot)
		c.Assert(err, tc.errChecker)
		if tc.errChecker == IsNil {
			c.Assert(ftm.numDeletes, Equals, tc.expNumDeletes)
		}
	}
}

func (s *VMWareSuite) TestGetSnapshotTags(c *C) {
	ctx := context.Background()
	for _, tc := range []struct {
		snapshotID   string
		catID        string
		categoryTags []vapitags.Tag
		expNumTags   int
		errGetTags   error
		errChecker   Checker
	}{
		{ // success
			snapshotID: "v1:s1",
			categoryTags: []vapitags.Tag{
				{Name: "v1:s1:t1:v1"},
				{Name: "v1:s1:t2:v2"},
				{Name: "v1:s2:t3:v3"},
				{Name: "v3:s2:t4:v4"},
			},
			errChecker: IsNil,
			catID:      "something",
			expNumTags: 2,
		},
		{ // bad tag
			snapshotID: "v1:s1",
			categoryTags: []vapitags.Tag{
				{Name: "v1:s1:t1:v1"},
				{Name: "v1:s1:t2:v2"},
				{Name: "v1:s2:t3:v3"},
				{Name: "v3:s2t4:v4"},
			},
			catID:      "something",
			errChecker: NotNil,
		},
		{ // bad tag
			snapshotID:   "v1:s1",
			categoryTags: []vapitags.Tag{},
			errGetTags:   errors.New("get tags error"),
			errChecker:   NotNil,
			catID:        "something",
		},
		{ // empty cat id
			errChecker: IsNil,
			catID:      "",
			expNumTags: 0,
		},
	} {
		ftm := &fakeTagManager{
			retGetTagsForCategory: tc.categoryTags,
			errGetTagsForCategory: tc.errGetTags,
		}
		provider := &FcdProvider{
			categoryID: tc.catID,
			tagManager: ftm,
		}
		tags, err := provider.getSnapshotTags(ctx, tc.snapshotID)
		c.Assert(err, tc.errChecker)
		if tc.errChecker == IsNil {
			c.Assert(len(tags), Equals, tc.expNumTags)
		}
	}
}

// An XML trace from `govc disk.snapshot.ls` with the VslmSyncFault
var (
	vslmSyncFaultReason = "Change tracking invalid or disk in use: api = DiskLib_BlockTrackGetEpoch, path->CValue() = /vmfs/volumes/vsan:52731cd109496ced-173f8e8aec7c6828/dc6d0c61-ec84-381f-2fa3-000c29e75b7f/4e1e7c4619a34919ae1f28fbb53fcd70-000008.vmdk"

	vslmSyncFaultReasonEsc = "Change tracking invalid or disk in use: api = DiskLib_BlockTrackGetEpoch, path-&gt;CValue() = /vmfs/volumes/vsan:52731cd109496ced-173f8e8aec7c6828/dc6d0c61-ec84-381f-2fa3-000c29e75b7f/4e1e7c4619a34919ae1f28fbb53fcd70-000008.vmdk"

	vslmSyncFaultString    = "A general system error occurred: " + vslmSyncFaultReason
	vslmSyncFaultStringEsc = "A general system error occurred: " + vslmSyncFaultReasonEsc

	vslmSyncFaultXML = `<Fault xmlns="http://schemas.xmlsoap.org/soap/envelope/">
	<faultcode>ServerFaultCode</faultcode>
	<faultstring>` + vslmSyncFaultStringEsc + `</faultstring>
	<detail>
	  <Fault xmlns:XMLSchema-instance="http://www.w3.org/2001/XMLSchema-instance" XMLSchema-instance:type="SystemError">
		<reason>` + vslmSyncFaultReasonEsc + `</reason>
	  </Fault>
	</detail>
	</Fault>`

	vslmSyncFaultXMLEnv = `<?xml version="1.0" encoding="UTF-8"?>
	<soapenv:Envelope xmlns:soapenc="http://schemas.xmlsoap.org/soap/encoding/"
	 xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/"
	 xmlns:xsd="http://www.w3.org/2001/XMLSchema"
	 xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
	<soapenv:Body>` + vslmSyncFaultXML + `</soapenv:Body>
	</soapenv:Envelope>`
)

func (s *VMWareSuite) TestFormatGovmomiError(c *C) {
	// basic soap fault
	fault := &soap.Fault{
		Code:   "soap-fault",
		String: "fault string",
	}
	soapFaultErr := soap.WrapSoapFault(fault)
	c.Assert(govmomiError{soapFaultErr}.Format(), Equals, "soap-fault: fault string")
	c.Assert(govmomiError{errors.Wrap(soapFaultErr, "outer wrapper")}.Format(), Equals, "outer wrapper: soap-fault: fault string")

	// Experiment with a real fault XML to figure out how to decode an error.
	// (adapted from govmomi/vim25/methods/fault_test.go)
	type TestBody struct {
		Fault *soap.Fault `xml:"http://schemas.xmlsoap.org/soap/envelope/ Fault,omitempty"`
	}
	body := TestBody{}
	env := soap.Envelope{Body: &body}
	dec := xml.NewDecoder(bytes.NewReader([]byte(vslmSyncFaultXMLEnv)))
	dec.TypeFunc = types.TypeFunc()
	err := dec.Decode(&env)
	c.Assert(err, IsNil)
	c.Assert(body.Fault, NotNil)

	err = soap.WrapSoapFault(body.Fault)
	c.Assert(soap.IsSoapFault(err), Equals, true)
	c.Assert(err.Error(), Equals, "ServerFaultCode: "+vslmSyncFaultString) // details present

	vimFault := &types.VimFault{
		MethodFault: types.MethodFault{
			FaultCause: &types.LocalizedMethodFault{
				LocalizedMessage: err.Error(),
			},
		},
	}
	err = soap.WrapVimFault(vimFault)
	c.Assert(soap.IsVimFault(err), Equals, true)
	c.Assert(err.Error(), Equals, "VimFault") // lost the details

	// A vslmFault fault with details such as that returned by gom.SnapshotCreate when
	// a volume CTK file is moved. (Note: govc succeeds in this case but list will fail)
	vslmFaultValue := "(vmodl.fault.SystemError) {\n   faultCause = null,\n   faultMessage = null,\n   reason = " + vslmSyncFaultReason + "}"
	vslmFault := &vslmtypes.VslmSyncFault{
		VslmFault: vslmtypes.VslmFault{
			MethodFault: types.MethodFault{
				FaultMessage: []types.LocalizableMessage{
					{
						Key: "com.vmware.pbm.pbmFault.locale",
						Arg: []types.KeyAnyValue{
							{
								Key:   "summary",
								Value: vslmFaultValue,
							},
						},
					},
				},
			},
		},
		Id: &types.ID{},
	}
	c.Assert(vslmFault.GetMethodFault(), NotNil)
	c.Assert(vslmFault.GetMethodFault().FaultMessage, DeepEquals, vslmFault.FaultMessage)

	err = soap.WrapVimFault(vslmFault)
	c.Assert(err.Error(), Equals, "VslmSyncFault")
	c.Assert(govmomiError{err}.Format(), Equals, "["+err.Error()+"; "+vslmFaultValue+"]")
	c.Assert(govmomiError{errors.Wrap(err, "outer wrapper")}.Format(), Equals, "[outer wrapper: "+err.Error()+"; "+vslmFaultValue+"]")

	c.Assert(govmomiError{err}.Matches(reVslmSyncFaultFatal), Equals, true)

	// task errors
	te := govmomitask.Error{
		LocalizedMethodFault: &types.LocalizedMethodFault{
			Fault: vslmFault,
		},
		Description: &types.LocalizableMessage{
			Message: "description message",
		},
	}
	c.Assert(err.Error(), Equals, "VslmSyncFault")
	c.Assert(govmomiError{te}.Format(), Equals, "[description message; "+vslmFaultValue+"]")
	c.Assert(govmomiError{errors.Wrap(te, "outer wrapper")}.Format(), Equals, "[outer wrapper: ; description message; "+vslmFaultValue+"]")

	// normal error
	testError := errors.New("test-error")
	c.Assert(govmomiError{testError}.Format(), Equals, testError.Error())

	// nil
	c.Assert(govmomiError{nil}.Format(), Equals, "")
}

type fakeTagManager struct {
	retGetTagsForCategory []vapitags.Tag
	errGetTagsForCategory error

	numDeletes   int
	errDeleteTag error

	numCreates   int
	errCreateTag error
}

func (f *fakeTagManager) GetCategory(ctx context.Context, id string) (*vapitags.Category, error) {
	return nil, nil
}
func (f *fakeTagManager) CreateCategory(ctx context.Context, category *vapitags.Category) (string, error) {
	return "", nil
}
func (f *fakeTagManager) CreateTag(ctx context.Context, tag *vapitags.Tag) (string, error) {
	f.numCreates++
	return "", f.errCreateTag
}
func (f *fakeTagManager) GetTagsForCategory(ctx context.Context, id string) ([]vapitags.Tag, error) {
	return f.retGetTagsForCategory, f.errGetTagsForCategory
}
func (f *fakeTagManager) DeleteTag(ctx context.Context, tag *vapitags.Tag) error {
	f.numDeletes++
	return f.errDeleteTag
}
