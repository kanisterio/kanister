package function

import (
	"github.com/kanisterio/kanister/pkg/param"

	"gopkg.in/check.v1"
)

type ScaleWorkloadSuite struct{}

var _ = check.Suite(&ScaleWorkloadSuite{})

func (s *ScaleWorkloadSuite) TestSetArgs(c *check.C) {
	stsParams := &param.StatefulSetParams{}
	for _, tc := range []struct {
		replicas         interface{}
		expectedReplicas int32
	}{
		{
			replicas:         4,
			expectedReplicas: 4,
		},
		{
			replicas:         234324,
			expectedReplicas: 234324,
		},
		{
			replicas:         234324,
			expectedReplicas: 234324,
		},
		{
			replicas:         2147483647,
			expectedReplicas: 2147483647, // 2147483647 is the maximum value int32 can hold
		},
	} {
		s := scaleWorkloadFunc{}
		s.setArgs(param.TemplateParams{
			StatefulSet: stsParams,
		}, map[string]interface{}{
			"replicas": tc.replicas,
		})

		c.Assert(s.replicas, check.Equals, tc.expectedReplicas)
	}
}
