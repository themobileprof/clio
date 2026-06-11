package intent

import "testing"

func TestDetectConversationalQueries(t *testing.T) {
	cases := []struct {
		query    string
		contains string
	}{
		{"what files are here", "ls"},
		{"where am I", "pwd"},
		{"how much memory do I have left", "free"},
		{"memory usage", "free"},
		{"disk usage", "df"},
		{"what is my ip address", "curl"},
		{"list running processes", "ps"},
		{"what processes are running on my phone", "ps"},
		{"show me the end of the log file", "tail"},
		{"view current directory", "pwd"},
		{"how do I unzip a zip file", "unzip"},
		{"change file permissions", "chmod"},
		{"chek disk space please", "df"},
		{"I want to see all the files in this folder", "ls"},
		{"how do I delete a directory", "rm"},
		{"can you help me find large files", "find"},
	}

	for _, c := range cases {
		result, err := Detect(c.query)
		if err != nil {
			t.Errorf("Detect(%q): unexpected error: %v", c.query, err)
			continue
		}
		if result.Source != "static" {
			t.Errorf("Detect(%q): source = %q, want static", c.query, result.Source)
		}
		if !containsStr(result.Command, c.contains) {
			t.Errorf("Detect(%q) = %q, want substring %q", c.query, result.Command, c.contains)
		}
	}
}

func containsStr(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (s == sub || len(sub) <= len(s) && findSub(s, sub)))
}

func findSub(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestSearchTextInFiles(t *testing.T) {
	result, err := Detect("search for text in files")
	if err != nil {
		t.Fatal(err)
	}
	if !containsStr(result.Command, "grep") {
		t.Fatalf("got %q, want grep", result.Command)
	}
}
