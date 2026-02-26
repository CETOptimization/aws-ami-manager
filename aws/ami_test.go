package aws

import (
	"testing"

	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestFormatTags(t *testing.T) {
	tests := []struct {
		name     string
		tags     *[]ec2Types.Tag
		expected string
	}{
		{
			name:     "nil tags",
			tags:     nil,
			expected: "<nil>",
		},
		{
			name:     "empty tags",
			tags:     &[]ec2Types.Tag{},
			expected: "<empty>",
		},
		{
			name: "single tag with valid values",
			tags: &[]ec2Types.Tag{
				{Key: strPtr("Name"), Value: strPtr("test-ami")},
			},
			expected: "Name=test-ami",
		},
		{
			name: "multiple tags",
			tags: &[]ec2Types.Tag{
				{Key: strPtr("Name"), Value: strPtr("test-ami")},
				{Key: strPtr("Environment"), Value: strPtr("prod")},
				{Key: strPtr("Team"), Value: strPtr("platform")},
			},
			expected: "Name=test-ami, Environment=prod, Team=platform",
		},
		{
			name: "tag with nil key",
			tags: &[]ec2Types.Tag{
				{Key: nil, Value: strPtr("value")},
			},
			expected: "<nil>=value",
		},
		{
			name: "tag with nil value",
			tags: &[]ec2Types.Tag{
				{Key: strPtr("Name"), Value: nil},
			},
			expected: "Name=<nil>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTags(tt.tags)
			if result != tt.expected {
				t.Errorf("formatTags() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestConvertTagSliceToMap(t *testing.T) {
	tests := []struct {
		name     string
		tags     []ec2Types.Tag
		expected map[string]ec2Types.Tag
	}{
		{
			name:     "nil slice",
			tags:     nil,
			expected: map[string]ec2Types.Tag{},
		},
		{
			name:     "empty slice",
			tags:     []ec2Types.Tag{},
			expected: map[string]ec2Types.Tag{},
		},
		{
			name: "single tag",
			tags: []ec2Types.Tag{
				{Key: strPtr("Name"), Value: strPtr("test-ami")},
			},
			expected: map[string]ec2Types.Tag{
				"Name": {Key: strPtr("Name"), Value: strPtr("test-ami")},
			},
		},
		{
			name: "multiple tags",
			tags: []ec2Types.Tag{
				{Key: strPtr("Name"), Value: strPtr("test-ami")},
				{Key: strPtr("Environment"), Value: strPtr("prod")},
			},
			expected: map[string]ec2Types.Tag{
				"Name":        {Key: strPtr("Name"), Value: strPtr("test-ami")},
				"Environment": {Key: strPtr("Environment"), Value: strPtr("prod")},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertTagSliceToMap(tt.tags)
			if len(result) != len(tt.expected) {
				t.Errorf("convertTagSliceToMap() length = %d, want %d", len(result), len(tt.expected))
				return
			}

			for key, expectedTag := range tt.expected {
				resultTag, ok := result[key]
				if !ok {
					t.Errorf("convertTagSliceToMap() missing key %q", key)
					continue
				}

				if (resultTag.Key == nil && expectedTag.Key != nil) ||
					(resultTag.Key != nil && expectedTag.Key == nil) ||
					(resultTag.Key != nil && *resultTag.Key != *expectedTag.Key) {
					t.Errorf("convertTagSliceToMap() key mismatch for %q", key)
				}

				if (resultTag.Value == nil && expectedTag.Value != nil) ||
					(resultTag.Value != nil && expectedTag.Value == nil) ||
					(resultTag.Value != nil && *resultTag.Value != *expectedTag.Value) {
					t.Errorf("convertTagSliceToMap() value mismatch for %q", key)
				}
			}
		})
	}
}

func TestConvertTagSliceToFilter(t *testing.T) {
	tests := []struct {
		name        string
		tags        []ec2Types.Tag
		expectedLen int
	}{
		{
			name:        "empty tags",
			tags:        []ec2Types.Tag{},
			expectedLen: 0,
		},
		{
			name: "single tag",
			tags: []ec2Types.Tag{
				{Key: strPtr("Name"), Value: strPtr("test-ami")},
			},
			expectedLen: 1,
		},
		{
			name: "multiple tags",
			tags: []ec2Types.Tag{
				{Key: strPtr("Name"), Value: strPtr("test-ami")},
				{Key: strPtr("Environment"), Value: strPtr("prod")},
			},
			expectedLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertTagSliceToFilter(tt.tags)
			if len(result) != tt.expectedLen {
				t.Errorf("convertTagSliceToFilter() length = %d, want %d", len(result), tt.expectedLen)
			}

			// Verify each filter has correct tag: prefix
			for i, filter := range result {
				if filter.Name == nil {
					t.Errorf("convertTagSliceToFilter() filter[%d] name is nil", i)
					continue
				}

				// Name should be "tag:KeyName"
				if len(*filter.Name) < 5 || (*filter.Name)[:4] != "tag:" {
					t.Errorf("convertTagSliceToFilter() filter[%d] name = %q, should start with 'tag:'", i, *filter.Name)
				}
			}
		})
	}
}

func TestCreateLaunchPermissionsForOwners(t *testing.T) {
	tests := []struct {
		name          string
		owners        []string
		expectedLen   int
		expectedFirst *string
	}{
		{
			name:        "nil owners",
			owners:      nil,
			expectedLen: 0,
		},
		{
			name:        "empty owners",
			owners:      []string{},
			expectedLen: 0,
		},
		{
			name:          "single owner",
			owners:        []string{"123456789012"},
			expectedLen:   1,
			expectedFirst: strPtr("123456789012"),
		},
		{
			name:          "multiple owners",
			owners:        []string{"123456789012", "987654321098", "111111111111"},
			expectedLen:   3,
			expectedFirst: strPtr("123456789012"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := createLaunchPermissionsForOwners(tt.owners)

			if len(result) != tt.expectedLen {
				t.Errorf("createLaunchPermissionsForOwners() length = %d, want %d", len(result), tt.expectedLen)
				return
			}

			if tt.expectedLen > 0 && tt.expectedFirst != nil {
				if result[0].UserId == nil || *result[0].UserId != *tt.expectedFirst {
					t.Errorf("createLaunchPermissionsForOwners() first owner = %v, want %v", result[0].UserId, tt.expectedFirst)
				}
			}

			// Verify no nil entries
			for i, perm := range result {
				if perm.UserId == nil {
					t.Errorf("createLaunchPermissionsForOwners() permission[%d] UserId is nil", i)
				}
			}
		})
	}
}

func TestNewAmi(t *testing.T) {
	amiID := "ami-0123456789abcdef0"
	ami := NewAmi(amiID)

	if ami.SourceAmiID != amiID {
		t.Errorf("NewAmi() SourceAmiID = %q, want %q", ami.SourceAmiID, amiID)
	}

	if ami.AmisPerRegion != nil {
		t.Error("NewAmi() AmisPerRegion should be nil for NewAmi()")
	}
}

func TestNewAmiWithRegions(t *testing.T) {
	amiID := "ami-0123456789abcdef0"
	sourceRegion := "us-east-1"
	regions := []string{"eu-west-1", "ap-southeast-1", "us-west-2"}

	ami := NewAmiWithRegions(amiID, sourceRegion, regions)

	if ami.SourceAmiID != amiID {
		t.Errorf("NewAmiWithRegions() SourceAmiID = %q, want %q", ami.SourceAmiID, amiID)
	}

	if ami.SourceRegion != sourceRegion {
		t.Errorf("NewAmiWithRegions() SourceRegion = %q, want %q", ami.SourceRegion, sourceRegion)
	}

	if len(ami.AmisPerRegion) != len(regions) {
		t.Errorf("NewAmiWithRegions() AmisPerRegion length = %d, want %d", len(ami.AmisPerRegion), len(regions))
	}

	for _, region := range regions {
		if _, ok := ami.AmisPerRegion[region]; !ok {
			t.Errorf("NewAmiWithRegions() missing region %q in AmisPerRegion", region)
		} else if ami.AmisPerRegion[region].SourceRegion != region {
			t.Errorf("NewAmiWithRegions() region %q has SourceRegion = %q", region, ami.AmisPerRegion[region].SourceRegion)
		}
	}
}

// Helper function to create string pointers for tests
func strPtr(s string) *string {
	return &s
}
