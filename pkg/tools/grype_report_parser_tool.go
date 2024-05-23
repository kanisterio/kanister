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
	Artifact        artifact            `json:"artifact"`
}

type fixVersionsResponse struct {
	Versions []string `json:"versions"`
	State    string   `json:"state"`
}

type vulnerabilityReport struct {
	ID          string              `json:"id"`
	DataSource  string              `json:"dataSource,omitempty"`
	Severity    string              `json:"severity"`
	Namespace   string              `json:"namespace"`
	Description string              `json:"description"`
	FixVersions fixVersionsResponse `json:"fix"`
}

type artifact struct {
	Name      string          `json:"name"`
	Version   string          `json:"version"`
	Type      string          `json:"type"`
	Purl      string          `json:"purl"`
	Locations json.RawMessage `json:"locations,omitempty"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
}

// filterVulnerabilityReportMatches filters vulnerabilities based on the severity levels set in severityTypeSet
func filterVulnerabilityReportMatches(matches []matchResponse, severityTypeSet map[string]bool) ([]matchResponse, error) {
	filtered := make([]matchResponse, 0)
	for _, m := range matches {
		if severityTypeSet[m.Vulnerabilities.Severity] {
			filtered = append(filtered, m)
		}
	}
	return filtered, nil
}

// decodeVulnerabilityReports unmarshals the specific matches from the vulnerability report
// and returns a list of vulnerabilities based on the severity levels set in severityTypeSet
func decodeVulnerabilityReports(v vulnerabilityScannerResponse, severityTypeSet map[string]bool) ([]matchResponse, error) {
	var mr []matchResponse
	if err := json.Unmarshal(v.Matches, &mr); err != nil {
		return make([]matchResponse, 0), fmt.Errorf("failed to unmarshal matches: %v", err)
	}
	return filterVulnerabilityReportMatches(mr, severityTypeSet)
}

// parseVulerabilitiesReport unmarshals the vulnerability report and returns a list of vulnerabilities
// based on the severity levels set in severityTypeSet
func parseVulerabilitiesReport(filePath string, severityLevels []string) ([]matchResponse, error) {
	mr := make([]matchResponse, 0)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return mr, fmt.Errorf("failed to read file at path %s: %v", filePath, err)
	}
	var response vulnerabilityScannerResponse

	if err = json.Unmarshal(data, &response); err != nil {
		return mr, fmt.Errorf("failed to unmarshal response: %v", err)
	}
	severityTypeSet := make(map[string]bool)
	for _, severityLevel := range severityLevels {
		severityTypeSet[severityLevel] = true
	}
	return decodeVulnerabilityReports(response, severityTypeSet)
}

// printResult Displays the filtered list of vulnerability reports to stdout
func printResult(mr []matchResponse, githubActionOutput bool) {
	for _, response := range mr {
		fmt.Printf("ID: %s\n", response.Vulnerabilities.ID)
		fmt.Printf("Link: %s\n", response.Vulnerabilities.DataSource)
		fmt.Printf("Severity: %s\n", response.Vulnerabilities.Severity)
		fmt.Printf("Namespace: %s\n", response.Vulnerabilities.Namespace)
		fmt.Printf("Description: %s\n", response.Vulnerabilities.Description)
		fmt.Printf("Fix Versions: %v\n", response.Vulnerabilities.FixVersions)
		fmt.Println("Package:")
		fmt.Printf("Name: %v\n", response.Artifact.Name)
		fmt.Printf("Version: %v\n", response.Artifact.Version)
		fmt.Printf("Type: %v\n", response.Artifact.Type)
		fmt.Printf("PURL: %v\n", response.Artifact.Purl)
		if githubActionOutput {
			fmt.Println("::group::Locations")
			fmt.Printf("%s\n", response.Artifact.Locations)
			fmt.Println("::endgroup::")
			fmt.Println("::group::Metadata")
			fmt.Printf("%s\n", response.Artifact.Metadata)
			fmt.Println("::endgroup::")
		} else {
			fmt.Printf("Locations: %s\n", response.Artifact.Locations)
			fmt.Printf("Metadata: \n%s\n", response.Artifact.Metadata)
		}
		fmt.Printf("\n")
	}
}

func main() {
	validSeverityLevels := []string{"Negliable", "Low", "Medium", "High", "Critical"}
	severityInputList := flag.String("s", "High,Critical", "Comma separated list of severity levels to scan. Valid severity levels are: "+strings.Join(validSeverityLevels, ","))
	githubActionOutput := flag.Bool("github", false, "Whether to use github action output format")
	reportJSONFilePath := flag.String("p", "", "Path to the JSON file containing the vulnerabilities report")
	flag.Parse()

	// passing file path is compulsory
	if *reportJSONFilePath == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}
	severityLevels := strings.Split(*severityInputList, ",")
	mr, err := parseVulerabilitiesReport(*reportJSONFilePath, severityLevels)
	if err != nil {
		fmt.Printf("Failed to parse vulnerabilities report: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Found %d vulnerabilities\n", len(mr))
	if len(mr) == 0 {
		os.Exit(0)
	}
	printResult(mr, *githubActionOutput)
	os.Exit(1)
}
