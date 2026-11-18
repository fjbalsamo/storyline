package adapters

import "encoding/json"

func quoteString(s string) string {
	b, _ := json.Marshal(s) // marshaling a string cannot fail
	return string(b)
}
