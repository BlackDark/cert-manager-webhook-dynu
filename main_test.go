package main

import (
	"testing"

	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid config",
			input:   `{"apiKeySecretRef":{"name":"my-secret","key":"api-key"}}`,
			wantErr: false,
		},
		{
			name:    "nil config",
			input:   "",
			wantErr: true,
		},
		{
			name:    "missing name",
			input:   `{"apiKeySecretRef":{"name":"","key":"api-key"}}`,
			wantErr: true,
		},
		{
			name:    "missing key",
			input:   `{"apiKeySecretRef":{"name":"my-secret","key":""}}`,
			wantErr: true,
		},
		{
			name:    "invalid json",
			input:   `not-json`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfgJSON *extapi.JSON
			if tt.input != "" {
				cfgJSON = &extapi.JSON{Raw: []byte(tt.input)}
			}
			_, err := loadConfig(cfgJSON)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
