package arm

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/go-test/deep"
)

func TestHashScaleSets(t *testing.T) {
	tests := []struct {
		name        string
		original    map[string]interface{}
		expectedErr error
		expectedTag string
	}{
		{
			name: "create a tag in master ss",
			original: map[string]interface{}{
				"resources": []interface{}{
					map[string]interface{}{
						"type": "Microsoft.Compute/virtualMachineScaleSets",
						"name": "ss-master",
						"tags": map[string]interface{}{},
					},
				},
			},
			expectedTag: "0sXC7FN0M4Bw2ZTTOoRL/8baXwfCAoM6jwa6KPlG138=",
		},
		{
			name: "update a tag in master ss",
			original: map[string]interface{}{
				"resources": []interface{}{
					map[string]interface{}{
						"type": "Microsoft.Compute/virtualMachineScaleSets",
						"name": "ss-master",
						"tags": map[string]interface{}{
							"scaleset-checksum": "blablabliblo",
						},
					},
				},
			},
			expectedTag: "0sXC7FN0M4Bw2ZTTOoRL/8baXwfCAoM6jwa6KPlG138=",
		},
		{
			name: "update a tag in infra ss",
			original: map[string]interface{}{
				"resources": []interface{}{
					map[string]interface{}{
						"type": "Microsoft.Compute/virtualMachineScaleSets",
						"name": "ss-infra",
						"tags": map[string]interface{}{
							"scaleset-checksum": "whatever",
						},
					},
				},
			},
			expectedTag: "qH5LdTkN5EPigRBlzw4vLTkHbxNhw524SVWRaMmrBGY=",
		},
		{
			name: "capacity does not affect hash",
			original: map[string]interface{}{
				"resources": []interface{}{
					map[string]interface{}{
						"type": "Microsoft.Compute/virtualMachineScaleSets",
						"name": "ss-infra",
						"tags": map[string]interface{}{
							"scaleset-checksum": "whatever",
						},
						"sku": map[string]interface{}{
							"capacity": "1",
						},
					},
				},
			},
			expectedTag: "ZGvsnKKMezqJ681Thjw4k2Xe+Jnjo6MOIoJoLKqjjLc=",
		},
		{
			name: "capacity does not affect hash (cont)",
			original: map[string]interface{}{
				"resources": []interface{}{
					map[string]interface{}{
						"type": "Microsoft.Compute/virtualMachineScaleSets",
						"name": "ss-infra",
						"tags": map[string]interface{}{
							"scaleset-checksum": "whatever",
						},
						"sku": map[string]interface{}{
							"capacity": "2",
						},
					},
				},
			},
			expectedTag: "ZGvsnKKMezqJ681Thjw4k2Xe+Jnjo6MOIoJoLKqjjLc=",
		},
	}

	for _, test := range tests {
		// Deep-copy original template so we can compare its diff after hashScaleSets finishes
		after := deepCopy(test.original)
		// Deep-copy once more for the actual test
		copied := deepCopy(test.original)
		if err := hashScaleSets(test.original, copied); !reflect.DeepEqual(err, test.expectedErr) {
			t.Errorf("%s: unexpected error: %v, expected %v", test.name, err, test.expectedErr)
			continue
		}
		if test.expectedTag != "" {
			// Remove expected tag from test.original
			compareAndRemove(t, test.expectedTag, test.original)
			compareAndRemove(t, "", after)

		}
		// Check that nothing else is missing from the original template
		if !reflect.DeepEqual(test.original, after) {
			t.Errorf("%s: unexpected diff:", test.name)
			for _, diff := range deep.Equal(test.original, after) {
				t.Errorf("- " + diff)
			}
		}
	}

}

func deepCopy(template map[string]interface{}) map[string]interface{} {
	data, err := json.Marshal(template)
	if err != nil {
		panic(err)
	}
	var copied map[string]interface{}
	if err := json.Unmarshal(data, &copied); err != nil {
		panic(err)
	}
	return copied
}

func compareAndRemove(t *testing.T, expected string, template map[string]interface{}) {
	for key, value := range template {
		if key != "resources" {
			continue
		}

		for _, r := range value.([]interface{}) {
			resource, ok := r.(map[string]interface{})
			if !ok {
				continue
			}

			if !isScaleSet(resource) {
				continue
			}

			// cleanup previous hash
			for k, v := range resource {
				if k != "tags" {
					continue
				}
				tags := v.(map[string]interface{})
				if expected != "" && tags[hashKey].(string) != expected {
					t.Errorf("unexpected tag: %q, expected: %q", tags[hashKey].(string), expected)
				}
				delete(tags, hashKey)
				resource[k] = tags
				break
			}
		}
	}
}
