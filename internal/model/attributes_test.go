package model

import "testing"

func TestOnlyVaultRAListsAreExtensible(t *testing.T) {
	extensibleUUIDs := map[string]struct{}{
		RA_PROFILE_ENGINE_ATTR: {},
		RA_PROFILE_ROLE_ATTR:   {},
	}
	seen := make(map[string]struct{})
	for _, attribute := range GetAttributeList() {
		dataAttribute, ok := attribute.(DataAttribute)
		if !ok {
			continue
		}
		if dataAttribute.Properties == nil {
			t.Fatalf("%s has nil properties", dataAttribute.Name)
		}
		_, want := extensibleUUIDs[dataAttribute.Uuid]
		got := dataAttribute.Properties.ExtensibleList
		if got != want {
			t.Errorf("%s extensibleList = %t, want %t", dataAttribute.Name, got, want)
		}
		if got {
			seen[dataAttribute.Uuid] = struct{}{}
		}
	}
	if len(seen) != len(extensibleUUIDs) {
		t.Fatalf("found %d extensible attributes, want %d", len(seen), len(extensibleUUIDs))
	}
}
