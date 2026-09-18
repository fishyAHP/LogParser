package tokenizer

import (
	"reflect"
	"testing"
)

func TestTokenizer_Tokenize(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		separator rune
		want      []Token
	}{
		{
			name:      "simple input",
			input:     "hello:world",
			separator: ':',
			want: []Token{
				{Value: "hello", Type: TokenString},
				{Value: ":", Type: TokenSeparator},
				{Value: "world", Type: TokenString},
				{Value: "", Type: TokenEOF},
			},
		},
		{
			name:      "multiple separators",
			input:     "a:b:c",
			separator: ':',
			want: []Token{
				{Value: "a", Type: TokenString},
				{Value: ":", Type: TokenSeparator},
				{Value: "b", Type: TokenString},
				{Value: ":", Type: TokenSeparator},
				{Value: "c", Type: TokenString},
				{Value: "", Type: TokenEOF},
			},
		},
		{
			name:      "no separator",
			input:     "hello",
			separator: ':',
			want: []Token{
				{Value: "hello", Type: TokenString},
				{Value: "", Type: TokenEOF},
			},
		},
		{
			name:      "separator at beginning",
			input:     ":hello",
			separator: ':',
			want: []Token{
				{Value: "", Type: TokenString},
				{Value: ":", Type: TokenSeparator},
				{Value: "hello", Type: TokenString},
				{Value: "", Type: TokenEOF},
			},
		},
		{
			name:      "separator at end",
			input:     "hello:",
			separator: ':',
			want: []Token{
				{Value: "hello", Type: TokenString},
				{Value: ":", Type: TokenSeparator},
				{Value: "", Type: TokenString},
				{Value: "", Type: TokenEOF},
			},
		},
		{
			name:      "two consecutive separators",
			input:     "a::b",
			separator: ':',
			want: []Token{
				{Value: "a", Type: TokenString},
				{Value: ":", Type: TokenSeparator},
				{Value: "", Type: TokenString},
				{Value: ":", Type: TokenSeparator},
				{Value: "b", Type: TokenString},
				{Value: "", Type: TokenEOF},
			},
		},
		{
			name:      "only separator",
			input:     ":",
			separator: ':',
			want: []Token{
				{Value: "", Type: TokenString},
				{Value: ":", Type: TokenSeparator},
				{Value: "", Type: TokenString},
				{Value: "", Type: TokenEOF},
			},
		},
		{
			name:      "empty input",
			input:     "",
			separator: ':',
			want: []Token{
				{Value: "", Type: TokenEOF},
			},
		},
		{
			name:      "unicode",
			input:     "привет:мир",
			separator: ':',
			want: []Token{
				{Value: "привет", Type: TokenString},
				{Value: ":", Type: TokenSeparator},
				{Value: "мир", Type: TokenString},
				{Value: "", Type: TokenEOF},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.separator)

			got := tokenizer.Tokenize(tt.input)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Tokenize() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
