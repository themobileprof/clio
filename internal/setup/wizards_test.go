package setup

import "testing"

func TestWizardByID(t *testing.T) {
	for _, id := range []string{"termux", "vim", "git", "devtools", "database"} {
		if WizardByID(id) == nil {
			t.Errorf("WizardByID(%q) = nil", id)
		}
	}
}
