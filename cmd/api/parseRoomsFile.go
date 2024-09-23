package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type EscapeRooms []struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func parseEscapeRooms(filename string) (EscapeRooms, error) {
	var rooms EscapeRooms

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	err = json.Unmarshal(data, &rooms)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return rooms, nil
}
