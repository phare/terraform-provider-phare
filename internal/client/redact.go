package client

import (
	"bytes"
	"encoding/json"
)

// redactedValue replaces the values of sensitive keys in logged bodies
const redactedValue = "***"

// sensitiveBodyKeys lists the JSON keys whose values are secrets. It must
// include the API field of every attribute marked as sensitive in the schemas.
var sensitiveBodyKeys = map[string]struct{}{
	"access_password":   {}, // status pages
	"access_token":      {}, // status pages, ntfy integrations
	"api_key":           {}, // pushover integrations
	"integration_key":   {}, // ilert integrations
	"secret":            {}, // outgoing webhook integrations
	"user_agent_secret": {}, // http monitors
	"user_key":          {}, // pushover integrations
}

// redactBody returns a request or response body for logging, with the values
// of sensitive keys replaced at any depth. Bodies that are not valid JSON are
// returned as is, the API only sends secrets in JSON bodies.
func redactBody(body []byte) string {
	decoder := json.NewDecoder(bytes.NewReader(body))
	// Keep numbers as is instead of converting them to float64
	decoder.UseNumber()

	var data any
	if err := decoder.Decode(&data); err != nil {
		return string(body)
	}

	if !redactValue(data) {
		return string(body)
	}

	redacted, err := json.Marshal(data)
	if err != nil {
		return string(body)
	}

	return string(redacted)
}

// redactValue replaces the values of sensitive keys in a decoded JSON value,
// and reports whether a value was replaced
func redactValue(value any) bool {
	redacted := false

	switch v := value.(type) {
	case map[string]any:
		for key, item := range v {
			if _, ok := sensitiveBodyKeys[key]; ok && item != nil {
				v[key] = redactedValue
				redacted = true
				continue
			}
			redacted = redactValue(item) || redacted
		}
	case []any:
		for _, item := range v {
			redacted = redactValue(item) || redacted
		}
	}

	return redacted
}
