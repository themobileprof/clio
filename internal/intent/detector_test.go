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

func TestDetectNigerianStudentQueries(t *testing.T) {
	cases := []struct {
		query    string
		contains string
	}{
		{"wetin dey inside this folder", "ls"},
		{"abeg show me the files here", "ls"},
		{"my phone storage don full", "df"},
		{"space don finish abeg", "df"},
		{"data no dey work", "ping"},
		{"internet no dey work o", "ping"},
		{"app dey jam", "kill"},
		{"app dey jam wetin i go do", "kill"},
		{"phone dey slow", "free"},
		{"e no gree again abeg", "kill"},
		{"comot this file abeg", "rm"},
		{"i wan install python for termux", "python"},
		{"how i go take clone project from github", "git clone"},
		{"abeg help me find lecture note pdf", "pdf"},
		{"make i create new folder for project", "mkdir"},
		{"send file to my laptop", "scp"},
		{"download something from the web", "wget"},
		{"update my termux packages abeg", "pkg"},
		{"commit my code make i push", "git push"},
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

func TestDetectTermuxSetup(t *testing.T) {
	queries := []string{
		"setup",
		"termux setup",
		"how do I setup termux for coding",
		"abeg help me setup termux",
	}
	for _, q := range queries {
		result, err := Detect(q)
		if err != nil {
			t.Errorf("Detect(%q): %v", q, err)
			continue
		}
		if result.Source != "setup" {
			t.Errorf("Detect(%q): source = %q, want setup", q, result.Source)
		}
		if !containsStr(result.Command, "termux_setup") {
			t.Errorf("Detect(%q) = %q, want termux_setup", q, result.Command)
		}
	}
}

func TestDetectSlangAndFuzzy(t *testing.T) {
	cases := []struct {
		query    string
		contains string
	}{
		{"my phone is acting somehow", "kill"},
		{"phone is lagging bad", "free"},
		{"chekc disk space abeg", "df"},
		{"storage don finish", "df"},
	}
	for _, c := range cases {
		result, err := Detect(c.query)
		if err != nil {
			t.Errorf("Detect(%q): %v", c.query, err)
			continue
		}
		if !containsStr(result.Command, c.contains) {
			t.Errorf("Detect(%q) = %q, want %q", c.query, result.Command, c.contains)
		}
	}
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
