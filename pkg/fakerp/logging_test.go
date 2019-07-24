package fakerp

import (
	"testing"
)

func TestParseLogAnalyticsResourceID(t *testing.T) {
	tests := []struct {
		resourceID string
		wantSub    string
		wantRG     string
		wantName   string
		wantErr    bool
	}{
		{
			resourceID: "/subscriptions/sub-1234/resourcegroups/defaultresourcegroup-x/providers/microsoft.operationalinsights/workspaces/DefaultWorkspace-x",
			wantSub:    "sub-1234",
			wantRG:     "defaultresourcegroup-x",
			wantName:   "DefaultWorkspace-x",
		},
		{
			resourceID: "some-id-09434",
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.resourceID, func(t *testing.T) {
			got, got1, got2, err := parseLogAnalyticsResourceID(tt.resourceID)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseLogAnalyticsResourceID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantSub {
				t.Errorf("parseLogAnalyticsResourceID() got = %v, want %v", got, tt.wantSub)
			}
			if got1 != tt.wantRG {
				t.Errorf("parseLogAnalyticsResourceID() got1 = %v, want %v", got1, tt.wantRG)
			}
			if got2 != tt.wantName {
				t.Errorf("parseLogAnalyticsResourceID() got2 = %v, want %v", got2, tt.wantName)
			}
		})
	}
}
