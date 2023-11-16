package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

type vulnerabilityScannerResponse struct {
	Matches                json.RawMessage `json:"matches"`
	RelatedVulnerabilities json.RawMessage `json:"relatedVulnerabilities"`
	MatchDetails           json.RawMessage `json:"matchDetails"`
	Artifact               json.RawMessage `json:"artifact"`
}

type matchResponse struct {
	Vulnerabilities vulnerabilityReport `json:"vulnerability"`
}

type fixVersionsResponse struct {
	Versions []string `json:"versions"`
	State    string   `json:"state"`
}

type vulnerabilityReport struct {
	ID          string              `json:"id"`
	Severity    string              `json:"severity"`
	Namespace   string              `json:"namespace"`
	Description string              `json:"description"`
	FixVersions fixVersionsResponse `json:"fix"`
}

// filterVulnerabilityReportMatches filters vulnerabilities based on the severity levels set in severityTypeSet
func filterVulnerabilityReportMatches(matches []matchResponse, severityTypeSet map[string]bool) ([]vulnerabilityReport, error) {
	mv := make([]vulnerabilityReport, 0)
	for _, m := range matches {
		if severityTypeSet[m.Vulnerabilities.Severity] {
			mv = append(mv, m.Vulnerabilities)
		}
	}
	return mv, nil
}

// decodeVulnerabilityReports unmarshals the specific matches from the vulnerability report
// and returns a list of vulnerabilities based on the severity levels set in severityTypeSet
func decodeVulnerabilityReports(v vulnerabilityScannerResponse, severityTypeSet map[string]bool) ([]vulnerabilityReport, error) {
	var mr []matchResponse
	mv := make([]vulnerabilityReport, 0)
	if err := json.Unmarshal(v.Matches, &mr); err != nil {
		return mv, fmt.Errorf("failed to unmarshal matches: %v", err)
	}
	return filterVulnerabilityReportMatches(mr, severityTypeSet)
}

// parseVulerabilitiesReport unmarshals the vulnerability report and returns a list of vulnerabilities
// based on the severity levels set in severityTypeSet
func parseVulerabilitiesReport(filePath string, severityLevels []string) ([]vulnerabilityReport, error) {
	mv := make([]vulnerabilityReport, 0)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return mv, fmt.Errorf("failed to read file at path %s: %v", filePath, err)
	}
	var response vulnerabilityScannerResponse

	if err = json.Unmarshal(data, &response); err != nil {
		return mv, fmt.Errorf("failed to unmarshal response: %v", err)
	}
	severityTypeSet := make(map[string]bool)
	for _, severityLevel := range severityLevels {
		severityTypeSet[severityLevel] = true
	}
	return decodeVulnerabilityReports(response, severityTypeSet)
}

// printResult Displays the filtered list of vulnerability reports to stdout
func printResult(mv []vulnerabilityReport) {
	for _, vulnerability := range mv {
		fmt.Printf("ID: %s\n", vulnerability.ID)
		fmt.Printf("Severity: %s\n", vulnerability.Severity)
		fmt.Printf("Namespace: %s\n", vulnerability.Namespace)
		fmt.Printf("Description: %s\n", vulnerability.Description)
		fmt.Printf("Fix Versions: %v\n", vulnerability.FixVersions)
		fmt.Printf("\n")
	}
}

func main() {
	validSeverityLevels := []string{"Negliable", "Low", "Medium", "High", "Critical"}
	severityInputList := flag.String("s", "High,Critical", "Comma separated list of severity levels to scan. Valid severity levels are: "+strings.Join(validSeverityLevels, ","))
	reportJsonFilePath := flag.String("p", "", "Path to the JSON file containing the vulnerabilities report")
	flag.Parse()

	// passing file path is compulsory
	if *reportJsonFilePath == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}
	severityLevels := strings.Split(*severityInputList, ",")
	mv, err := parseVulerabilitiesReport(*reportJsonFilePath, severityLevels)
	if err != nil {
		fmt.Printf("Failed to parse vulnerabilities report: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Found %d vulnerabilities\n", len(mv))
	if len(mv) == 0 {
		os.Exit(0)
	}
	printResult(mv)
	os.Exit(1)
}
