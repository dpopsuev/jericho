package billing

import (
	"fmt"
	"strings"
	"time"
)

// fmtTokens formats a token count with K/M suffix for readability.
func fmtTokens(n int) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000.0)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000.0)
	}
	return fmt.Sprintf("%d", n)
}

// fmtDuration formats a duration as "Xm Ys" or "Ys".
func fmtDuration(d time.Duration) string {
	s := int(d.Seconds())
	if s >= 60 {
		return fmt.Sprintf("%dm %ds", s/60, s%60)
	}
	return fmt.Sprintf("%ds", s)
}

// truncate shortens s to maxLen characters, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// fmtMDTable renders a simple GitHub-flavoured Markdown table.
// header is the header row, rows are the data rows, and footer is an
// optional final row (pass nil to omit).
func fmtMDTable(header []string, rows [][]string, footer []string) string {
	var b strings.Builder

	// Header.
	b.WriteString("| ")
	b.WriteString(strings.Join(header, " | "))
	b.WriteString(" |\n")

	// Separator.
	b.WriteString("|")
	for range header {
		b.WriteString(" --- |")
	}
	b.WriteString("\n")

	// Data rows.
	for _, row := range rows {
		b.WriteString("| ")
		b.WriteString(strings.Join(row, " | "))
		b.WriteString(" |\n")
	}

	// Footer row.
	if len(footer) > 0 {
		b.WriteString("| ")
		b.WriteString(strings.Join(footer, " | "))
		b.WriteString(" |\n")
	}

	return b.String()
}
