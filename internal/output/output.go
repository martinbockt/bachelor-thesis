package output

import (
	"fmt"
	"os"
	"time"

	"github.com/gocarina/gocsv"
)

type Information struct {
	ID                   int           `csv:"ID"`
	LLM                  string        `csv:"LLM"`
	LLMDuration          time.Duration `csv:"LLM Duration"`
	RequestDuration      time.Duration `csv:"Request Duration"`
	WebsitesChecked      int           `csv:"Websites Checked"`
	WebsiteMaxLength     int           `csv:"Website Max Length"`
	WebsiteReducedLength int           `csv:"Website Reduced Length"`
	TokenCount           int           `csv:"Token Count"`
	ProviderURL          string        `csv:"Provider URL"`
	ProviderName         string        `csv:"Provider Name"`
	RoomName             string        `csv:"Room Name"`
	Description          string        `csv:"Description"`
	MinPlayers           int           `csv:"Min Players"`
	MaxPlayers           int           `csv:"Max Players"`
	Duration             int           `csv:"Duration"`
	BookingURL           string        `csv:"Booking URL"`
	DetailPageURL        string        `csv:"Detail Page URL"`
	ImageURL             string        `csv:"Image URL"`
	Genre                string        `csv:"Genre"`
	Difficulty           string        `csv:"Difficulty"`
	TokenLimitReached    bool          `csv:"Token Limit Reached"`
	Error                string        `csv:"Error"`
}

type Output struct {
	information []Information
}

func New() *Output {
	return &Output{}
}

func (o *Output) AddInformation(i Information) {
	for _, info := range o.information {
		if info.RoomName != i.RoomName || info.ID != i.ID || info.LLM != i.LLM || info.DetailPageURL != i.DetailPageURL {
			continue
		}
		tokenCount := info.TokenCount
		info = i
		if info.TokenCount == 0 {
			info.TokenCount = tokenCount
		}

		return
	}
	o.information = append(o.information, i)
}

func (o *Output) SaveAsCSV(llmName string) error {
	file, err := os.Create(llmName + "output.csv")
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Marshal the information slice to the file in CSV format
	if err := gocsv.MarshalFile(&o.information, file); err != nil {
		return fmt.Errorf("failed to marshal information to file: %w", err)
	}

	return nil
}

func (o *Output) ReadOutputCSV(llmName string) ([]Information, error) {
	file, err := os.Open(llmName + "output.csv")
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var information []Information
	if err := gocsv.UnmarshalFile(file, &information); err != nil {
		return nil, fmt.Errorf("failed to unmarshal file to information: %w", err)
	}

	o.information = information

	return information, nil
}
