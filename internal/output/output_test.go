package output_test

import (
	"time"

	"github.com/martinbockt/esc-llm-webscraper/internal/output"
)

func ExampleOutput_AddInformation() {
	// Create a new output
	o := output.New()

	// Create a new information struct
	i := output.Information{
		LLM:                  "test",
		LLMDuration:          time.Second,
		RequestDuration:      time.Second,
		WebsitesChecked:      1,
		WebsiteMaxLength:     1,
		WebsiteReducedLength: 1,
		ProviderURL:          "test",
		ProviderName:         "test",
		RoomName:             "test",
		MinPlayers:           1,
		MaxPlayers:           1,
		Duration:             1,
		BookingURL:           "test",
		DetailPageURL:        "test",
		ImageURL:             "test",
		Genre:                "test",
		Difficulty:           "test",
		Error:                "test",
	}

	// Add the information to the output
	o.AddInformation(i)

	o.SaveAsCSV("")
	// Output:
}
