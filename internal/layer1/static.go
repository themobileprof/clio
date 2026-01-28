package layer1

import (
    "strings"
)

// CommandEntry holds the command and a description
type CommandEntry struct {
    Cmd  string
    Desc string
}

// VerbNounCatalog is a map of Verb -> Noun -> CommandEntry
var VerbNounCatalog = map[string]map[string]CommandEntry{
    "list": {
        "file":      {"ls -F", "List files in current directory"},
        "files":     {"ls -F", "List files in current directory"},
        "directory": {"ls -d */", "List directories only"},
        "folder":    {"ls -d */", "List directories only"},
        "all":       {"ls -la", "List all files including hidden ones"},
        "hidden":    {"ls -la", "List all files including hidden ones"},
        "content":   {"ls -la", "List contents"},
        "permission": {"ls -l", "List files with permissions"},
        "process":   {"ps aux", "List running processes"},
        "job":       {"jobs", "List background jobs"},
        "alias":     {"alias", "List current aliases"},
        "variable":  {"printenv", "List environment variables"},
        "env":       {"printenv", "List environment variables"},
        "disk":      {"df -h", "List disk space usage"},
        "space":     {"df -h", "List disk space usage"},
        "memory":    {"free -h", "List memory usage"},
        "ram":       {"free -h", "List memory usage"},
        "ip":        {"ip a", "List IP addresses"},
        "network":   {"ifconfig", "List network interfaces"},
        "route":     {"ip route", "List network routes"},
        "user":      {"cat /etc/passwd", "List system users"},
        "group":     {"cat /etc/group", "List system groups"},
        "history":   {"history", "List command history"},
        "module":    {"npm list", "List npm modules (if applicable)"}, // Context aware?
        "package":   {"pkg list-installed", "List installed packages (Termux)"},
        "port":      {"netstat -tuln", "List open ports"},
        "connection": {"netstat -an", "List network connections"},
        "usb":       {"lsusb", "List USB devices"},
        "cpu":       {"lscpu", "List CPU info"},
        "hardware":  {"lshw", "List hardware configuration"},
    },
    "create": {
        "directory": {"mkdir -p", "Create a new directory"},
        "folder":    {"mkdir -p", "Create a new directory"},
        "file":      {"touch", "Create an empty file"},
        "link":      {"ln -s", "Create a symbolic link"},
        "symlink":   {"ln -s", "Create a symbolic link"},
        "alias":     {"alias name='cmd'", "Create an alias"},
        "user":      {"useradd", "Create a new user"},
        "group":     {"groupadd", "Create a new group"},
        "password":  {"passwd", "Create/Change password"},
        "key":       {"ssh-keygen", "Create SSH key pair"},
        "archive":   {"tar -czvf archive.tar.gz", "Create a compressed archive"},
        "zip":       {"zip -r archive.zip", "Create a zip archive"},
    },
    "remove": {
        "file":      {"rm", "Remove a file"},
        "directory": {"rm -rf", "Remove a directory recursively"},
        "folder":    {"rm -rf", "Remove a directory recursively"},
        "all":       {"rm -rf *", "Remove everything in current directory"},
        "process":   {"kill", "Kill a process"},
        "package":   {"pkg uninstall", "Uninstall a package"},
        "user":      {"userdel", "Remove a user"},
        "group":     {"groupdel", "Remove a group"},
        "permission": {"chmod -x", "Remove execution permission"},
        "alias":     {"unalias", "Remove an alias"},
        "job":       {"kill %1", "Kill a background job"},
    },
    "copy": {
        "file":      {"cp", "Copy a file"},
        "directory": {"cp -r", "Copy a directory recursively"},
        "folder":    {"cp -r", "Copy a directory recursively"},
        "content":   {"cp -r", "Copy contents"},
        "remote":    {"scp", "Copy files to/from remote host"},
        "text":      {"xclip -sel clip", "Copy text to clipboard (if installed)"},
    },
    "move": {
        "file":      {"mv", "Move or rename a file"},
        "directory": {"mv", "Move or rename a directory"},
        "folder":    {"mv", "Move or rename a directory"},
    },
    "rename": {
        "file":      {"mv", "Rename a file"},
        "directory": {"mv", "Rename a directory"},
        "folder":    {"mv", "Rename a directory"},
    },
    "view": {
        "file":      {"cat", "View file content"},
        "content":   {"cat", "View file content"},
        "log":       {"tail -f", "View log file in real-time"},
        "end":       {"tail", "View end of file"},
        "start":     {"head", "View start of file"},
        "process":   {"top", "View running processes"},
        "tree":      {"tree", "View directory tree"},
        "calendar":  {"cal", "View calendar"},
        "date":      {"date", "View current date and time"},
        "path":      {"pwd", "View current working directory"},
    },
    "edit": {
        "file":      {"nano", "Edit file with nano"},
        "code":      {"vim", "Edit file with vim"},
        "permission": {"chmod", "Change file permissions"},
        "owner":     {"chown", "Change file owner"},
        "group":     {"chgrp", "Change file group"},
        "password":  {"passwd", "Change user password"},
    },
    "search": {
        "file":      {"find . -name", "Search for files by name"},
        "text":      {"grep -r", "Search for text in files"},
        "string":    {"grep -r", "Search for text in files"},
        "command":   {"which", "Locate a command"},
        "package":   {"pkg search", "Search for packages"},
        "history":   {"history | grep", "Search command history"},
        "process":   {"pgrep", "Search for process ID"},
    },
    "check": {
        "disk":      {"df -h", "Check disk space"},
        "space":     {"df -h", "Check disk space"},
        "memory":    {"free -h", "Check memory usage"},
        "ram":       {"free -h", "Check memory usage"},
        "cpu":       {"lscpu", "Check CPU info"},
        "network":   {"ping google.com", "Check network connectivity"},
        "internet":  {"ping google.com", "Check internet connectivity"},
        "port":      {"netstat -tuln", "Check open ports"},
        "version":   {"uname -a", "Check kernel version"},
        "os":        {"cat /etc/os-release", "Check OS details"},
        "ip":        {"curl ifconfig.me", "Check public IP"},
    },
    "install": {
        "package":   {"pkg install", "Install a package"},
        "software":  {"pkg install", "Install software"},
        "update":    {"pkg update", "Update package list"},
        "upgrade":   {"pkg upgrade", "Upgrade installed packages"},
        "git":       {"pkg install git", "Install git"},
        "python":    {"pkg install python", "Install python"},
        "node":      {"pkg install nodejs", "Install nodejs"},
    },
    "run": {
        "script":    {"./script.sh", "Run a shell script"},
        "python":    {"python script.py", "Run a python script"},
        "background": {"&", "Run command in background"},
        "admin":     {"sudo", "Run as root/admin"},
    },
    "archive": {
        "file":      {"tar -czvf", "Compress files into tar.gz"},
        "directory": {"tar -czvf", "Compress directory into tar.gz"},
    },
    "extract": {
        "file":      {"tar -xzvf", "Extract tar.gz archive"},
        "archive":   {"tar -xzvf", "Extract tar.gz archive"},
        "zip":       {"unzip", "Extract zip archive"},
    },
    "download": {
        "file":      {"wget", "Download file from URL"},
        "web":       {"curl -O", "Download file from URL"},
    },
}

// Aliases for verbs to normalize input
var VerbAliases = map[string]string{
    "show": "list", "see": "list", "display": "list", "ls": "list", "dir": "list",
    "make": "create", "new": "create", "generate": "create", "touch": "create",
    "delete": "remove", "del": "remove", "erase": "remove", "rm": "remove", "kill": "remove",
    "duplicate": "copy", "clone": "copy", "cp": "copy",
    "mv": "move", "transfer": "move",
    "name": "rename",
    "read": "view", "open": "view", "cat": "view", "watch": "view",
    "modify": "edit", "change": "edit", "write": "edit",
    "find": "search", "locate": "search", "query": "search", "lookup": "search",
    "verify": "check", "test": "check", "monitor": "check",
    "get": "install", "add": "install",
    "execute": "run", "start": "run", "launch": "run",
    "compress": "archive", "zip": "archive", "pack": "archive",
    "unzip": "extract", "unpack": "extract",
    "wait": "check",
}

// NounAliases for normalization
var NounAliases = map[string]string{
    "dirs": "directory", "folders": "directory", "dir": "directory",
    "doc": "file", "docs": "file", "document": "file",
    "program": "process", "app": "process", "task": "process",
    "img": "file", "image": "file", "video": "file",
    "connection": "network", "wifi": "network", "net": "network",
    "storage": "disk", "hdd": "disk", "ssd": "disk",
    "usage": "memory",
    "addr": "ip", "address": "ip",
}

// PreProcess now returns a structured parsed intent
func ParseIntent(input string) (string, string) {
    tokens := strings.Fields(strings.ToLower(input))
    if len(tokens) == 0 {
        return "", ""
    }
    
    // Basic stemming and alias resolution
    var verb, noun string
    
    // Attempt to identify verb (usually first word, or first significant word)
    for _, t := range tokens {
        stemmed := Stem(t)
        if v, ok := VerbAliases[stemmed]; ok {
            verb = v
            break
        }
        if _, ok := VerbNounCatalog[stemmed]; ok {
            verb = stemmed
            break
        }
    }
    
    // Attempt to identify noun (any word after verb ideally, or anywhere)
    for _, t := range tokens {
        stemmed := Stem(t)
        // Skip if it's the verb we just found
        if stemmed == verb || VideoAliases[stemmed] == verb { // typo in check, but logic holds
             continue 
        }
        
        if n, ok := NounAliases[stemmed]; ok {
            noun = n
            break 
        }
        // Check if this noun exists in the verb's map
        if verb != "" {
            if _, ok := VerbNounCatalog[verb][stemmed]; ok {
                noun = stemmed
                break
            }
        }
    }
    
    // Default fallback if only verb found
    if verb != "" && noun == "" {
        // Can we guess a default noun? 
        // e.g. "list" -> "file" matches "ls"
        if _, ok := VerbNounCatalog[verb]["file"]; ok {
            return verb, "file"
        }
    }

    return verb, noun
}

// Stem is a very simple stemmer for common suffixes
func Stem(word string) string {
    word = strings.TrimSpace(word)
    if strings.HasSuffix(word, "ing") {
        return word[:len(word)-3]
    }
    if strings.HasSuffix(word, "ed") {
        return word[:len(word)-2]
    }
    if strings.HasSuffix(word, "s") && !strings.HasSuffix(word, "ss") {
        return word[:len(word)-1]
    }
    return word
}

// VideoAliases was a typo in logic above, fixed logic is to use VerbAliases map.
var VideoAliases = VerbAliases // Just for compilation safety if I referenced it
