package utils

import "fmt"

type indicator string

const (
	Fail indicator = `❌`
	Pass indicator = `✅`
	Skip indicator = `🚫`
)

func PrintStage(description string, i indicator) {
	switch i {
	case Pass:
		fmt.Printf("Passed the '%s' check.. %s\n", description, i)
	case Skip:
		fmt.Printf("Skipping the '%s' check.. %s\n", description, i)
	case Fail:
		fmt.Printf("Failed the '%s' check.. %s\n", description, i)
	default:
		fmt.Println(description)
	}
}
