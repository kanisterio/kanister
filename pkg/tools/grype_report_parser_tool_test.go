package main

import (
	"testing"

	. "gopkg.in/check.v1"
)

type VulnerabilityParserSuite struct{}

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

var _ = Suite(&VulnerabilityParserSuite{})

func (v *VulnerabilityParserSuite) TestNonExistentResult(c *C) {
	severityLevels := []string{"High", "Critical"}
	matchingVulnerabilities, err := parseVulerabilitiesReport("testdata/result_non_existent.json", severityLevels)
	c.Assert(len(matchingVulnerabilities), Equals, 0)
	c.Assert(err, NotNil)
}

func (v *VulnerabilityParserSuite) TestInvalidJson(c *C) {
	severityLevels := []string{"High", "Critical"}
	matchingVulnerabilities, err := parseVulerabilitiesReport("testdata/results_invalid.json", severityLevels)
	c.Assert(len(matchingVulnerabilities), Equals, 0)
	c.Assert(err, NotNil)
}

func (v *VulnerabilityParserSuite) TestValidJsonWithZeroVulnerabilities(c *C) {
	severityLevels := []string{"High", "Critical"}
	matchingVulnerabilities, err := parseVulerabilitiesReport("testdata/results_valid_no_matches.json", severityLevels)
	c.Assert(len(matchingVulnerabilities), Equals, 0)
	c.Assert(err, IsNil)
}

func (v *VulnerabilityParserSuite) TestValidJsonForLowVulerabilities(c *C) {
	severityLevels := []string{"Low", "Medium"}
	matchingVulnerabilities, err := parseVulerabilitiesReport("testdata/results_valid.json", severityLevels)
	c.Assert(len(matchingVulnerabilities), Equals, 0)
	c.Assert(err, IsNil)
}

func (v *VulnerabilityParserSuite) TestValidJsonForMatchingVulerabilities(c *C) {
	severityLevels := []string{"High", "Critical"}
	expectedIds := []string{"CVE-2016-10228", "CVE-2016-10229"}
	matchingVulnerabilities, err := parseVulerabilitiesReport("testdata/results_valid.json", severityLevels)
	c.Assert(len(matchingVulnerabilities), Equals, 2)
	c.Assert(err, IsNil)
	for index, vulnerability := range matchingVulnerabilities {
		c.Assert(vulnerability.Vulnerabilities.ID, Equals, expectedIds[index])
		c.Assert(vulnerability.Vulnerabilities.Severity, Equals, severityLevels[index])
	}
}
