package layer1

import (
    "testing"
)

func TestStem(t *testing.T) {
    cases := []struct {
        Input    string
        Expected string
    }{
        {"listing", "list"},
        {"copied", "copy"}, // our simple stemmer handles 'ed' -> trim 2
        {"files", "file"},
        {"running", "run"},
        {"processes", "process"},
        {"status", "status"}, // "status" is now preserved as exception
        // Wait, logic: HasSuffix("s") && !HasSuffix("ss") -> trim 1.
        // "status" -> "statu".
    }

    for _, c := range cases {
        got := Stem(c.Input)
        if got != c.Expected {
            t.Errorf("Stem(%q) == %q, expected %q", c.Input, got, c.Expected)
        }
    }
}

func TestParseIntent(t *testing.T) {
    cases := []struct {
        Input        string
        ExpectedVerb string
        ExpectedNoun string
    }{
        {"list files", "list", "file"},
        {"listing files", "list", "file"}, // stemming check
        {"create directory", "create", "directory"},
        {"make folder", "create", "directory"}, // alias check
        {"copy doc", "copy", "file"}, // noun alias check
        {"delete images", "remove", "file"}, // alias + noun alias
        {"unknown command", "", ""},
        {"justlist", "", ""}, 
    }

    for _, c := range cases {
        v, n := ParseIntent(c.Input)
        if v != c.ExpectedVerb || n != c.ExpectedNoun {
            t.Errorf("ParseIntent(%q) == (%q, %q), expected (%q, %q)", 
                c.Input, v, n, c.ExpectedVerb, c.ExpectedNoun)
        }
    }
}
