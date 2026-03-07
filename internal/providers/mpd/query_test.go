package main

import (
	"testing"

	"github.com/fhs/gompd/v2/mpd"
)

func TestGetField(t *testing.T) {
	tests := []struct {
		name     string
		m        map[string]string
		keys     []string
		expected string
	}{
		{"first key exists", map[string]string{"Title": "Song1"}, []string{"Title", "title"}, "Song1"},
		{"second key exists", map[string]string{"title": "Song2"}, []string{"Title", "title"}, "Song2"},
		{"none exist", map[string]string{"Artist": "Art1"}, []string{"Title", "title"}, ""},
		{"empty value ignored", map[string]string{"Title": "  ", "title": "Song3"}, []string{"Title", "title"}, "Song3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetField(tt.m, tt.keys...); got != tt.expected {
				t.Errorf("GetField() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNormalizeFields(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"lowercase conversion", "TEST", "test"},
		{"replace ampersand", "Rock & Roll", "rock and roll"},
		{"remove punctuation", "Hello, World!", "hello world"},
		{"multiple spaces", "too   much  space", "too much space"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeFields(tt.input); got != tt.expected {
				t.Errorf("NormalizeFields() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCalculateScore(t *testing.T) {
	tests := []struct {
		name     string
		res      mpd.Attrs
		query    string
		expected int32
	}{
		{"exact match", mpd.Attrs{"Title": "test song"}, "test song", 500},
		{"prefix match", mpd.Attrs{"Title": "test song part 2"}, "test", 250},
		{"contains match", mpd.Attrs{"Title": "my test song"}, "test", 128},
		{"no match", mpd.Attrs{"Title": "unrelated"}, "test", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CalculateScore(tt.res, tt.query); got != tt.expected {
				t.Errorf("CalculateScore() = %v, want %v", got, tt.expected)
			}
		})
	}
}
