package main

import (
	"testing"

	"gopkg.in/check.v1"
)

type VulnerabilityParserSuite struct{}

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(&VulnerabilityParserSuite{})

func (v *VulnerabilityParserSuite) TestNonExistentResult(c *check.C) {
	severityLevels := []string{"High", "Critical"}
	matchingVulnerabilities, err := parseVulerabilitiesReport("testdata/result_non_existent.json", severityLevels)
	c.Assert(len(matchingVulnerabilities), check.Equals, 0)
	c.Assert(err, check.NotNil)
}

func (v *VulnerabilityParserSuite) TestInvalidJson(c *check.C) {
	severityLevels := []string{"High", "Critical"}
	matchingVulnerabilities, err := parseVulerabilitiesReport("testdata/results_invalid.json", severityLevels)
	c.Assert(len(matchingVulnerabilities), check.Equals, 0)
	c.Assert(err, check.NotNil)
}

func (v *VulnerabilityParserSuite) TestValidJsonWithZeroVulnerabilities(c *check.C) {
	severityLevels := []string{"High", "Critical"}
	matchingVulnerabilities, err := parseVulerabilitiesReport("testdata/results_valid_no_matches.json", severityLevels)
	c.Assert(len(matchingVulnerabilities), check.Equals, 0)
	c.Assert(err, check.IsNil)
}

func (v *VulnerabilityParserSuite) TestValidJsonForLowVulerabilities(c *check.C) {
	severityLevels := []string{"Low", "Medium"}
	matchingVulnerabilities, err := parseVulerabilitiesReport("testdata/results_valid.json", severityLevels)
	c.Assert(len(matchingVulnerabilities), check.Equals, 0)
	c.Assert(err, check.IsNil)
}

func (v *VulnerabilityParserSuite) TestValidJsonForMatchingVulerabilities(c *check.C) {
	severityLevels := []string{"High", "Critical"}
	expectedIds := []string{"CVE-2016-10228", "CVE-2016-10229"}
	matchingVulnerabilities, err := parseVulerabilitiesReport("testdata/results_valid.json", severityLevels)
	c.Assert(len(matchingVulnerabilities), check.Equals, 2)
	c.Assert(err, check.IsNil)
	for index, vulnerability := range matchingVulnerabilities {
		c.Assert(vulnerability.Vulnerabilities.ID, check.Equals, expectedIds[index])
		c.Assert(vulnerability.Vulnerabilities.Severity, check.Equals, severityLevels[index])
	}
}
