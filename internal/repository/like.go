package repository

import "strings"

// escapeLike neutralises LIKE wildcards in user input (PostgreSQL's default escape char is \).
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
