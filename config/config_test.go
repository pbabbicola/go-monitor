package config_test

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pbabbicola/go-monitor/config"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     []config.SiteElement
		wantErr  bool // Here I wanted to check for error.Is, but regexp/syntax does not use sentinel errors
	}{
		{
			name:     "happy path",
			filename: "testdata/correct.json",
			want: []config.SiteElement{
				{
					URL:             "https://duckduckgo.com",
					Regexp:          regexp.MustCompile("duck"),
					IntervalSeconds: 5,
				},
			},
			wantErr: false,
		},
		{
			name:     "bad regexp",
			filename: "testdata/bad_regexp.json",
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "bad json",
			filename: "testdata/bad_json.json",
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "file does not exist",
			filename: "testdata/does_not_exist.json",
			want:     nil,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := config.Parse(tt.filename)
			assert.Truef(t, err != nil == tt.wantErr, "wanted err to be %v, but got error %v", tt.wantErr, err)
			assert.Equal(t, got, tt.want)
		})
	}
}
