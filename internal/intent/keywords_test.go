package intent

import (
    "reflect"
    "testing"
)

func TestIsolateKeywords(t *testing.T) {
    cases := []struct {
        Input    string
        Expected []string
    }{
        {"How do I copy a file?", []string{"cp", "file"}}, // "copy" mapped to "cp"
        {"List all running processes", []string{"ls", "all", "run", "process"}},
        {"duplicate directory", []string{"cp", "directory"}},
    }

    for _, c := range cases {
        got := IsolateKeywords(c.Input)
        // Checking slice equality
        if !reflect.DeepEqual(got, c.Expected) {
             t.Errorf("IsolateKeywords(%q) == %v, expected %v", c.Input, got, c.Expected)
        }
    }
}
