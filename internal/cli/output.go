package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
)

// OutputMode represents the output format.
type OutputMode int

const (
	OutputNormal OutputMode = iota
	OutputMinimal
	OutputTable
	OutputJSON
)

var outputMode = OutputNormal

// SetOutputMode sets the global output mode.
func SetOutputMode(mode OutputMode) {
	outputMode = mode
}

// GetOutputMode returns the current output mode.
func GetOutputMode() OutputMode {
	if JSONOutput() {
		return OutputJSON
	}
	return outputMode
}

// Table provides a simple table formatter.
type Table struct {
	w       *tabwriter.Writer
	headers []string
}

// NewTable creates a new table with the given headers.
func NewTable(headers ...string) *Table {
	t := &Table{
		w:       tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0),
		headers: headers,
	}
	if len(headers) > 0 {
		_, _ = t.w.Write([]byte(strings.Join(headers, "\t") + "\n"))
	}
	return t
}

// NewTableWriter creates a table writing to a specific writer.
func NewTableWriter(out io.Writer, headers ...string) *Table {
	t := &Table{
		w:       tabwriter.NewWriter(out, 0, 0, 2, ' ', 0),
		headers: headers,
	}
	if len(headers) > 0 {
		_, _ = t.w.Write([]byte(strings.Join(headers, "\t") + "\n"))
	}
	return t
}

// Row adds a row to the table.
func (t *Table) Row(values ...string) {
	_, _ = t.w.Write([]byte(strings.Join(values, "\t") + "\n"))
}

// Flush writes the table output.
func (t *Table) Flush() {
	_ = t.w.Flush()
}

// Minimal prints minimal output (just the essential value).
func Minimal(value string) {
	fmt.Println(value)
}

// MinimalF prints minimal formatted output.
func MinimalF(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

// Normal prints normal output with a label.
func Normal(label, value string) {
	fmt.Printf("%s: %s\n", label, value)
}

// NormalF prints normal formatted output.
func NormalF(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

// StatusIcon returns an icon for the given boolean status.
func StatusIcon(active bool) string {
	if active {
		return "●"
	}
	return "○"
}

// TruncateString truncates a string to maxLen, adding "..." if truncated.
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// FormatDuration formats a duration in seconds as mm:ss or hh:mm:ss.
func FormatDuration(seconds int) string {
	if seconds < 0 {
		seconds = 0
	}
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60

	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

// FormatProgress formats a progress bar.
func FormatProgress(current, total int, width int) string {
	if total <= 0 {
		return strings.Repeat("─", width)
	}

	percent := float64(current) / float64(total)
	filled := int(percent * float64(width))
	if filled > width {
		filled = width
	}

	return strings.Repeat("━", filled) + strings.Repeat("─", width-filled)
}
