package parser

// StripComment removes a `%%` to-end-of-line comment. The `%%` only
// starts a comment when at the start of the line or preceded by
// whitespace, so tokens like "50%%" stay intact.
func StripComment(line string) string {
	for i := 0; i+1 < len(line); i++ {
		if line[i] != '%' || line[i+1] != '%' {
			continue
		}
		if i == 0 || line[i-1] == ' ' || line[i-1] == '\t' {
			return line[:i]
		}
	}
	return line
}
