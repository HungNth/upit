package app

import (
	"reflect"
	"testing"
)

func TestCountInputPlaceholders(t *testing.T) {
	tests := map[string]struct {
		data map[string]any
		want int
	}{
		"one nested value": {
			data: map[string]any{"target": map[string]any{"url": "{input}"}},
			want: 1,
		},
		"substring is not a placeholder": {
			data: map[string]any{"target": "prefix-{input}"},
			want: 0,
		},
		"object key is not a placeholder": {
			data: map[string]any{"{input}": "literal"},
			want: 0,
		},
		"multiple values": {
			data: map[string]any{"first": "{input}", "nested": []any{"{input}"}},
			want: 2,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if got := countInputPlaceholders(test.data); got != test.want {
				t.Errorf("countInputPlaceholders() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestReplaceInputPlaceholderPreservesTemplateShape(t *testing.T) {
	data := map[string]any{
		"target": map[string]any{"url": "{input}"},
		"metadata": []any{
			map[string]any{"label": "prefix-{input}"},
			true,
		},
	}
	replaceInputPlaceholder(data, "https://files.example.test/original")

	want := map[string]any{
		"target": map[string]any{"url": "https://files.example.test/original"},
		"metadata": []any{
			map[string]any{"label": "prefix-{input}"},
			true,
		},
	}
	if !reflect.DeepEqual(data, want) {
		t.Errorf("data = %#v, want %#v", data, want)
	}
}
