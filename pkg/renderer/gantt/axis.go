package gantt

import (
	"strconv"
	"strings"
	"time"
)

// d3StrftimeToGoLayout translates a d3-time-format strftime spec
// (the language Mermaid's `axisFormat` directive accepts) into a
// Go reference-time layout string. Covers the documented Mermaid
// token set:
//
//	%Y year (2006)        %y year, 2-digit (06)
//	%m month, 2-digit     %B month name (January)  %b month abbr (Jan)
//	%d day-of-month       %a weekday abbr (Mon)    %A weekday full (Monday)
//	%H hour 00-23         %I hour 01-12            %p AM/PM
//	%M minute             %S second                %L millisecond (000)
//	%j day-of-year        %% literal '%'
//
// Unknown tokens are passed through verbatim so a typo surfaces in
// the rendered axis rather than being silently swallowed.
func d3StrftimeToGoLayout(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '%' || i+1 >= len(s) {
			b.WriteByte(c)
			continue
		}
		i++
		switch s[i] {
		case 'Y':
			b.WriteString("2006")
		case 'y':
			b.WriteString("06")
		case 'm':
			b.WriteString("01")
		case 'B':
			b.WriteString("January")
		case 'b':
			b.WriteString("Jan")
		case 'd':
			b.WriteString("02")
		case 'e':
			b.WriteString("_2")
		case 'a':
			b.WriteString("Mon")
		case 'A':
			b.WriteString("Monday")
		case 'H':
			b.WriteString("15")
		case 'I':
			b.WriteString("03")
		case 'p':
			b.WriteString("PM")
		case 'M':
			b.WriteString("04")
		case 'S':
			b.WriteString("05")
		case 'L':
			b.WriteString("000")
		case 'j':
			b.WriteString("002")
		case '%':
			b.WriteByte('%')
		default:
			b.WriteByte('%')
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// tickStep represents one parsed `tickInterval` spec. Stepping
// uses time.AddDate for calendar units (day/week/month/year) so a
// `1month` tick lands on the same day-of-month rather than every
// 30 days, matching mmdc behaviour.
type tickStep struct {
	n       int
	unit    string // "millisecond" | "second" | "minute" | "hour" | "day" | "week" | "month" | "year"
	asDur   time.Duration
	addDate func(t time.Time, n int) time.Time
}

func (ts tickStep) advance(t time.Time) time.Time {
	if ts.addDate != nil {
		return ts.addDate(t, ts.n)
	}
	return t.Add(ts.asDur)
}

// parseTickInterval understands the Mermaid forms `<N><unit>` (no
// space) and `<N> <unit>` (with space). Returns ok=false when the
// spec is empty or malformed; callers fall back to the auto-chosen
// interval.
func parseTickInterval(s string) (tickStep, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return tickStep{}, false
	}
	i := 0
	for i < len(s) && (s[i] >= '0' && s[i] <= '9') {
		i++
	}
	if i == 0 {
		return tickStep{}, false
	}
	n, err := strconv.Atoi(s[:i])
	if err != nil || n <= 0 {
		return tickStep{}, false
	}
	unit := strings.TrimSpace(s[i:])
	switch unit {
	case "millisecond", "milliseconds":
		return tickStep{n: n, unit: "millisecond", asDur: time.Duration(n) * time.Millisecond}, true
	case "second", "seconds":
		return tickStep{n: n, unit: "second", asDur: time.Duration(n) * time.Second}, true
	case "minute", "minutes":
		return tickStep{n: n, unit: "minute", asDur: time.Duration(n) * time.Minute}, true
	case "hour", "hours":
		return tickStep{n: n, unit: "hour", asDur: time.Duration(n) * time.Hour}, true
	case "day", "days":
		return tickStep{n: n, unit: "day", addDate: func(t time.Time, d int) time.Time { return t.AddDate(0, 0, d) }}, true
	case "week", "weeks":
		return tickStep{n: n, unit: "week", addDate: func(t time.Time, w int) time.Time { return t.AddDate(0, 0, 7*w) }}, true
	case "month", "months":
		return tickStep{n: n, unit: "month", addDate: func(t time.Time, m int) time.Time { return t.AddDate(0, m, 0) }}, true
	case "year", "years":
		return tickStep{n: n, unit: "year", addDate: func(t time.Time, y int) time.Time { return t.AddDate(y, 0, 0) }}, true
	}
	return tickStep{}, false
}
