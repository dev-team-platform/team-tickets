package service

import (
	"reflect"
	"testing"
)

func TestScrubNestedJSONSecrets(t *testing.T) {
	jsonValue := `{"name":"safe","password":"bad","child":{"access_token":"bad","value":"ok"},"items":[{"otp":"bad","public":"yes"}]}`
	got := scrub(&jsonValue)
	want := map[string]interface{}{
		"name": "safe",
		"child": map[string]interface{}{
			"value": "ok",
		},
		"items": []interface{}{
			map[string]interface{}{
				"public": "yes",
			},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected redaction: %#v", got)
	}
}
