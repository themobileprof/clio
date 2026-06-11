#!/usr/bin/env bash
# Test natural-language query variations against clio pipe mode.
set -uo pipefail

CLIO="${CLIO:-/tmp/clio}"
PASS=0
FAIL=0
SKIP=0

# query|expected_substring (command output must contain this)
readarray -t CASES <<'EOF'
# list / ls
list files|ls
show all files|ls
display hidden files|ls
what files are here|ls
list running processes|ps
show disk space|df
list open ports|netstat
# copy / move / remove
copy a file|cp
duplicate file|cp
move file|mv
rename directory|mv
delete file|rm
remove folder|rm
how do I delete a directory|rm
# view
view file contents|cat
show end of log|tail
view current directory|pwd
# search
find pdf files|find
search for text in files|grep
locate a command|which
# check system
check disk space|df
how much disk space|df
check memory usage|free
memory usage|free
show ram usage|free
check cpu|lscpu
check network|ping
what is my ip|curl
# archives
extract tar file|tar
unzip file|unzip
create tar archive|tar
compress directory|tar
# permissions / edit
change file permissions|chmod
edit file|nano
# download
download file|wget
# install (termux-oriented)
install package|pkg
# run
run as admin|sudo
EOF

run_query() {
    local query="$1"
    local expected="$2"
    local out
    out=$(echo "$query" | "$CLIO" 2>/dev/null) || true

    if [[ -z "$out" ]]; then
        echo "FAIL  | $query"
        echo "       expected containing: $expected"
        echo "       got: (no match / exit error)"
        ((FAIL++)) || true
        return
    fi

    if [[ "$out" == *"$expected"* ]]; then
        echo "PASS  | $query"
        echo "       -> $out"
        ((PASS++)) || true
    else
        echo "FAIL  | $query"
        echo "       expected containing: $expected"
        echo "       got: $out"
        ((FAIL++)) || true
    fi
}

echo "Clio query variation test"
echo "Binary: $CLIO"
echo "========================================"
echo ""

for case in "${CASES[@]}"; do
    [[ -z "$case" || "$case" == \#* ]] && continue
    query="${case%%|*}"
    expected="${case#*|}"
    run_query "$query" "$expected"
    echo ""
done

echo "========================================"
echo "Results: $PASS passed, $FAIL failed"
exit $([[ $FAIL -eq 0 ]] && echo 0 || echo 1)
