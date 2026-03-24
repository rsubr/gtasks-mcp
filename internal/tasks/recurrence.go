package tasks

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const recurrenceMarkerPrefix = "\n\n<!-- gtasks-mcp:"
const recurrenceMarkerSuffix = " -->"

type recurrenceMetadata struct {
	Recurrence string `json:"recurrence,omitempty"`
}

func normalizeRecurrence(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func validRecurrence(value string) bool {
	switch normalizeRecurrence(value) {
	case "", "daily", "weekly", "monthly", "yearly":
		return true
	default:
		return false
	}
}

func splitNotesAndRecurrence(notes string) (string, string) {
	start := strings.LastIndex(notes, recurrenceMarkerPrefix)
	if start == -1 {
		return notes, ""
	}

	end := strings.LastIndex(notes, recurrenceMarkerSuffix)
	if end == -1 || end < start {
		return notes, ""
	}

	end += len(recurrenceMarkerSuffix)
	marker := notes[start:end]
	payload := strings.TrimSuffix(strings.TrimPrefix(marker, recurrenceMarkerPrefix), recurrenceMarkerSuffix)

	var meta recurrenceMetadata
	if err := json.Unmarshal([]byte(payload), &meta); err != nil {
		return notes, ""
	}

	baseNotes := strings.TrimSuffix(notes[:start], "\n")
	return baseNotes, normalizeRecurrence(meta.Recurrence)
}

func composeNotes(notes, recurrence string) string {
	recurrence = normalizeRecurrence(recurrence)
	if recurrence == "" {
		return notes
	}

	meta, _ := json.Marshal(recurrenceMetadata{Recurrence: recurrence})
	if strings.TrimSpace(notes) == "" {
		return recurrenceMarkerPrefix + string(meta) + recurrenceMarkerSuffix
	}

	return notes + recurrenceMarkerPrefix + string(meta) + recurrenceMarkerSuffix
}

func nextDueDate(due, recurrence string, now time.Time) (string, error) {
	recurrence = normalizeRecurrence(recurrence)
	base := now.UTC()
	if strings.TrimSpace(due) != "" {
		parsed, err := time.Parse(time.RFC3339, due)
		if err != nil {
			return "", fmt.Errorf("invalid due date for recurring task: %w", err)
		}
		base = parsed.UTC()
	}

	switch recurrence {
	case "daily":
		base = base.AddDate(0, 0, 1)
	case "weekly":
		base = base.AddDate(0, 0, 7)
	case "monthly":
		base = base.AddDate(0, 1, 0)
	case "yearly":
		base = base.AddDate(1, 0, 0)
	default:
		return "", fmt.Errorf("unsupported recurrence %q", recurrence)
	}

	return base.Format(time.RFC3339), nil
}
