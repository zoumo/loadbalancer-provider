package arp

import "testing"

func Test_loadCache(t *testing.T) {
	tests := []struct {
		name    string
		want    Caches
		wantErr bool
	}{
		{
			"loadcache", nil, false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadCache()
			if (err != nil) != tt.wantErr {
				t.Errorf("loadCache() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
