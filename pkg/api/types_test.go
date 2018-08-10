package api

import (
	"testing"
)

func TestTotalNodes(t *testing.T) {
	cases := []struct {
		p        Properties
		expected int
	}{
		{
			p: Properties{
				AgentPoolProfiles: []*AgentPoolProfile{
					{
						Count: 1,
					},
				},
			},
			expected: 1,
		},
		{
			p: Properties{
				AgentPoolProfiles: []*AgentPoolProfile{
					{
						Count: 3,
					},
					{
						Count: 4,
					},
				},
			},
			expected: 7,
		},
	}

	for _, c := range cases {
		if c.p.TotalNodes() != c.expected {
			t.Fatalf("expected TotalNodes() to return %d but instead returned %d", c.expected, c.p.TotalNodes())
		}
	}
}

func TestIsCustomVNET(t *testing.T) {
	cases := []struct {
		p              Properties
		expectedMaster bool
		expectedAgent  bool
	}{
		{
			p: Properties{
				AgentPoolProfiles: []*AgentPoolProfile{
					{
						VnetSubnetID: "testSubnet",
					},
				},
			},
			expectedMaster: true,
			expectedAgent:  true,
		},
		{
			p: Properties{
				AgentPoolProfiles: []*AgentPoolProfile{
					{
						Count: 1,
					},
					{
						Count: 1,
					},
				},
			},
			expectedMaster: false,
			expectedAgent:  false,
		},
	}

	for _, c := range cases {
		if c.p.AgentPoolProfiles[0].IsCustomVNET() != c.expectedAgent {
			t.Fatalf("expected IsCustomVnet() to return %t but instead returned %t", c.expectedAgent, c.p.AgentPoolProfiles[0].IsCustomVNET())
		}
	}

}
