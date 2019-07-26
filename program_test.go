package nordlead3

import (
	"testing"
)

func TestSetCategory(t *testing.T) {
	p := Program{category: 0x00}

	// Test standard case
	err := p.SetCategory(0x0B) // Synth
	if err != nil {
		t.Errorf("Setting category to %x failed unexpectedly: %s", 0x0B, err)
		return
	}
	if p.category != 0x0B {
		t.Errorf("Failed to set category. Expected %x got %x", 0x0B, p.category)
		return
	}

	// Test error handling
	err = p.SetCategory(0x0F) // invalid category
	if err != ErrorInvalidCategory {
		t.Errorf("Did not get expected error setting invalid category")
	}
	if p.category == 0x0F {
		t.Errorf("Incorrectly set category to invalid value")
	}
}

func TestSetName(t *testing.T) {
	var oldNameBytes [16]byte
	oldname := "ASixteenCharName"
	copy(oldNameBytes[:], oldname)
	p := Program{name: oldNameBytes}

	var expectedName [16]byte
	newName := "FooBar"
	copy(expectedName[:], newName)

	// Test standard case
	err := p.SetName("FooBar")
	if err != nil {
		t.Errorf("Setting name to %q failed unexpectedly: %s", "FooBar", err)
	}
	if p.name != expectedName {
		t.Errorf("Incorrectly set name. Expected %q got %q", expectedName, p.name)
	}

	// Test error handling

	// Test too long
	err = p.SetName("ANameThatIsWayTOOLong!")
	if err != ErrorInvalidName {
		t.Errorf("Did not get expected error setting too long name")
		return
	}
	if p.name != expectedName {
		t.Errorf("Incorrectly set name to invalid value")
	}

	// Test blank/empty
	err = p.SetName("")
	if err != ErrorInvalidName {
		t.Errorf("Did not get expected error setting blank name")
		return
	}
	if p.category == 0x0F {
		t.Errorf("Incorrectly set name to invalid value")
	}
}

func TestPrintableName(t *testing.T) {

}

func TestPrintableCategory(t *testing.T) {

}
