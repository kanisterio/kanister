package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
)

type VulnerabilityScannerResponse struct {
	Matches                json.RawMessage `json:"matches"`
	RelatedVulnerabilities json.RawMessage `json:"relatedVulnerabilities"`
	MatchDetails           json.RawMessage `json:"matchDetails"`
	Artifact               json.RawMessage `json:"artifact"`
}

type MatchResponse struct {
	Vulnerabilities VulnerabilityReport `json:"vulnerability"`
}

type FixVersionsResponse struct {
	Versions []string `json:"versions"`
	State    string   `json:"state"`
}

type VulnerabilityReport struct {
	Id          string              `json: id`
	Severity    string              `json:"severity"`
	Namespace   string              `json:"namespace"`
	Description string              `json:"description"`
	FixVersions FixVersionsResponse `json:"fix"`
}

// Filters vulnerabilites based on the severity levels set in severityTypeSet
func filterVulnerabilityReportMatches(matches []MatchResponse, severityTypeSet map[string]bool) ([]VulnerabilityReport, error) {
	matchingVulnerabilities := make([]VulnerabilityReport, 0)
	for _, match := range matches {
		if severityTypeSet[match.Vulnerabilities.Severity] {
			matchingVulnerabilities = append(matchingVulnerabilities, match.Vulnerabilities)
		}
	}
	return matchingVulnerabilities, nil
}

// Unmarshalls the Matches from the vulnerability report and returns a list of vulnerabilities
// based on the severity levels set in severityTypeSet
func decodeVulnerabilityReports(vulnerabilityScannerResponse VulnerabilityScannerResponse, severityTypeSet map[string]bool) ([]VulnerabilityReport, error) {
	var matches []MatchResponse
	matchingVulnerabilities := make([]VulnerabilityReport, 0)
	if err := json.Unmarshal(vulnerabilityScannerResponse.Matches, &matches); err != nil {
		return matchingVulnerabilities, errors.New("Error while unmarshalling a MatchResponse with err: " + err.Error())
	}
	return filterVulnerabilityReportMatches(matches, severityTypeSet)
}

// Unmarshalls the MatchDetails from the vulnerability report and returns a list of vulnerabilities
// based on the severity levels set in severityTypeSet
func parseVulerabilitiesReport(filePath string, severityLevels []string) ([]VulnerabilityReport, error) {
	matchingVulnerabilities := make([]VulnerabilityReport, 0)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return matchingVulnerabilities, errors.New("Error while reading file at path: " + filePath + " err: " + err.Error())
	}
	var vulnerabilityScannerResponse VulnerabilityScannerResponse

	if err = json.Unmarshal(data, &vulnerabilityScannerResponse); err != nil {
		return matchingVulnerabilities, errors.New("Error while parsing file at path: " + filePath + " err: " + err.Error())
	}
	severityTypeSet := make(map[string]bool)
	for _, severityLevel := range severityLevels {
		severityTypeSet[severityLevel] = true
	}
	return decodeVulnerabilityReports(vulnerabilityScannerResponse, severityTypeSet)
}

func main() {
	validSeverityLevels := []string{"Negliable", "Low", "Medium", "High", "Critical"}
	severityInputList := flag.String("sl", "High,Critical", "Comma separated list of severity levels to scan. Valid severity levels are: "+strings.Join(validSeverityLevels, ","))
	reportJsonFilePath := flag.String("p", "", "Path to the JSON file containing the vulnerabilities report")
	flag.Parse()

	// passing file path is compulsory
	if *reportJsonFilePath == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}
	severityLevels := strings.Split(*severityInputList, ",")
	matchingVulnerabilities, err := parseVulerabilitiesReport(*reportJsonFilePath, severityLevels)
	if err != nil {
		fmt.Printf("Found an error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Found %d vulnerabilities\n", len(matchingVulnerabilities))
	if len(matchingVulnerabilities) == 0 {
		os.Exit(0)
	}
	for _, vulnerability := range matchingVulnerabilities {
		fmt.Printf("Id: %s\n", vulnerability.Id)
		fmt.Printf("Severity: %s\n", vulnerability.Severity)
		fmt.Printf("Namespace: %s\n", vulnerability.Namespace)
		fmt.Printf("Description: %s\n", vulnerability.Description)
		fmt.Printf("Fix Versions: %v\n", vulnerability.FixVersions)
		fmt.Printf("\n")
	}
	os.Exit(1)
}
