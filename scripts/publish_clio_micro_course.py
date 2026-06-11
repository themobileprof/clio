#!/usr/bin/env python3
"""Publish the Clio micro course to TheMobileProf LMS (admin API)."""

from __future__ import annotations

import json
import os
import sys
import urllib.error
import urllib.request
from pathlib import Path

GEMINI_LESSONS_JSON = Path(__file__).resolve().parent / "clio_course_lessons_gemini.json"


def load_lessons() -> list[dict]:
    """Prefer Gemini-expanded lessons when the JSON export exists."""
    if GEMINI_LESSONS_JSON.exists():
        data = json.loads(GEMINI_LESSONS_JSON.read_text(encoding="utf-8"))
        lessons = data.get("lessons", [])
        if lessons:
            return [
                {
                    "title": L["title"],
                    "description": L["description"],
                    "content": L["content"],
                    "durationMinutes": L["durationMinutes"],
                }
                for L in lessons
            ]
    return LESSONS_OUTLINE


COURSE = {
    "title": "Clio: Your Terminal Assistant",
    "description": (
        "Learn to use Clio — the offline-first CLI assistant that turns plain English "
        "into shell commands. Built for Termux and Nigerian students who want to code "
        "on their phone without memorizing Linux syntax."
    ),
    "topic": "Linux",
    "duration": "90 min",
    "price": 15,
    "difficulty": "beginner",
    "isPublished": True,
    "objectives": (
        "Install Clio, ask questions in plain English and Pidgin, run setup wizards, "
        "download automation modules, and work confidently on Termux."
    ),
    "prerequisites": "Basic familiarity with opening a terminal (Termux or Linux).",
    "tags": ["clio", "termux", "linux", "cli", "nigeria", "terminal"],
}

LESSONS_OUTLINE = [
    {
        "title": "Lesson 1 — What Is Clio?",
        "description": "Install Clio and understand what it does on your phone or laptop.",
        "durationMinutes": 10,
        "content": """# What Is Clio?

**Clio** is your terminal assistant. Instead of memorizing commands like `tar -xzvf` or `df -h`, you type what you want in plain English — and Clio suggests the right command.

## Why students use it

- Works **offline first** — common tasks match instantly without internet
- Understands **Nigerian Pidgin** — "abeg show me files for downloads folder"
- Built for **Termux on Android** — low memory, no crashes
- Shows commands **before** running them — you stay in control

## Install on Termux or Linux

```bash
curl -fsSL https://clipilot.themobileprof.com/clio | sh
```

Then start it:

```bash
clio
```

You should see a welcome line and a `>>` prompt.

## Your first queries

Try these at the `>>` prompt:

```
>> list files in this folder
>> check disk space
>> what is my current directory
```

Clio shows a suggested command and asks what you want to do next.

## Key idea

Clio is **not** a chatbot that runs mystery code. It is a **command finder** with optional setup wizards and downloadable workflows.

## Checklist

- [ ] Clio installed and launches
- [ ] You asked at least one question in plain English
- [ ] You saw the confirm-before-run menu
""",
    },
    {
        "title": "Lesson 2 — Ask in Plain English",
        "description": "Use natural language, Pidgin, and full sentences to find shell commands.",
        "durationMinutes": 12,
        "content": """# Ask in Plain English

Clio matches your **intent**, not exact syntax.

## Shell commands (instant catalog)

These map to everyday Linux commands:

| You type | Clio suggests |
|----------|---------------|
| list files | `ls` |
| check disk space | `df -h` |
| find large files | `find . -type f -size +100M` |
| copy file | `cp` |
| check memory | `free -h` |

## Full sentences work

```
>> how do I see which processes are running
>> I want to extract a zip file
>> show me files modified in the last week
```

## Nigerian Pidgin & campus talk

Clio understands casual phrasing:

```
>> wetin dey inside this folder
>> abeg help me check memory
>> I wan install python for termux
```

## The result menu

When Clio finds a match you see:

1. **Show examples** — common flags and usage
2. **Run the command** — executes after you confirm
3. **Search again** — try another phrasing

Always read the command before pressing yes.

## Match types (shown in the header)

- **⌨️ SHELL COMMAND** — run directly in terminal
- **📦 AUTOMATION MODULE** — downloaded workflow (Lesson 4)
- **⭐ SETUP WIZARD** — install/configure tools (Lesson 3)

## Practice

Ask five questions mixing formal English and Pidgin. Note which type of match you get.
""",
    },
    {
        "title": "Lesson 3 — Setup Wizards",
        "description": "Use first-class setup wizards to configure Termux, Vim, Git, languages, and databases.",
        "durationMinutes": 15,
        "content": """# Setup Wizards

**Setup wizards** are for **installing and configuring** your environment — usually once.

They are **not** the same as automation modules (Lesson 4).

## How to open them

```
>> setup
```

Or pick one directly:

```
>> setup termux
>> setup vim
>> setup git
>> setup devtools
>> setup database
```

Natural language also works:

```
>> configure my dev environment
>> install git and gh
>> setup vim plugins
```

## Available wizards

| Wizard | Purpose |
|--------|---------|
| `setup termux` | Full phone dev environment (Zsh, storage, essentials) |
| `setup vim` | Vim + dev plugins |
| `setup git` | Git identity + GitHub CLI (`gh`) |
| `setup devtools` | Pick Python, Node, Go, PHP, Rust |
| `setup database` | Choose PostgreSQL, MariaDB, Redis, SQLite |

## How wizards run on Termux

Wizards execute via a helper script (not inside the Go binary):

```bash
clio-run-module termux_setup setup
```

Clio downloads the module from the CLIPilot server the first time you need it.

## Label to look for

When a wizard matches your question you see:

**⭐ [SETUP WIZARD]**

## Practice

1. Run `setup` and read the wizard list
2. Pick one wizard relevant to you
3. Run the `clio-run-module` command it prints (when online)
""",
    },
    {
        "title": "Lesson 4 — Automation Modules",
        "description": "Download and run repeatable task workflows from the CLIPilot registry.",
        "durationMinutes": 12,
        "content": """# Automation Modules

**Automation modules** are **repeatable workflows** — things you do often while working.

Examples: copy files, check disk space, backup a folder, monitor logs.

## Setup wizards vs automation modules

| | Setup wizards | Automation modules |
|---|---------------|-------------------|
| **When** | Once / occasionally | Whenever needed |
| **Purpose** | Install & configure | Do a task |
| **Browse** | `setup` | `modules` |
| **Label** | `[SETUP WIZARD]` | `[AUTOMATION]` |

## Browse modules

```
>> modules
```

Or see both types separated:

```
>> catalog
```

Each entry shows:

- Description from the server
- `download <module_id>` — fetch one module
- `clio-run-module <module_id> setup` — run it

## Download modules

**One module:**

```
>> download copy_file
```

**Bulk sync:**

```
>> sync
```

On Termux lite profile, `sync` fetches essential modules only. Use `sync full` for the complete catalog.

## Natural language

You can also ask directly:

```
>> check disk space
>> backup this directory
```

If a module matches, you see **📦 [AUTOMATION MODULE]** in the header.

## Practice

1. Run `modules` and pick one that interests you
2. `download <id>` then run the printed `clio-run-module` command
""",
    },
    {
        "title": "Lesson 5 — Offline Mode & Sync",
        "description": "Understand lite profile, layers, and when you need internet.",
        "durationMinutes": 10,
        "content": """# Offline Mode & Sync

Clio is **offline first**. Most common questions never touch the network.

## Detection layers (simplified)

1. **Phrase & catalog** — instant, built into the binary
2. **Man pages** — searches local manuals (skipped on lite profile)
3. **Automation modules** — local SQLite cache
4. **Remote search** — CLIPilot API when local match fails

## Lite profile (Termux / low RAM)

On phones Clio often runs in **lite** mode:

- Smaller memory footprint
- `sync` downloads essential modules only
- Use `sync full` when you want everything

Config lives at `~/.clio/config.yaml`:

```yaml
profile: auto   # auto | lite | full
remote_search: auto
```

## When you need internet

- First-time **module download** (`download`, `sync`, or setup wizard)
- **Remote search** when local layers cannot match your question
- Installing Clio itself (`curl ... | sh`)

## Local storage

- Database: `~/.clio/clio.db` (cached modules)
- Config: `~/.clio/config.yaml`

## Practice

1. Turn off Wi‑Fi and ask three common questions — they should still work
2. Run `sync` when back online
""",
    },
    {
        "title": "Lesson 6 — Termux on Android",
        "description": "Phone-specific tips: install, clio-run-module, storage, and 32-bit devices.",
        "durationMinutes": 12,
        "content": """# Termux on Android

Clio is designed for students coding on their phone.

## Install on Termux

```bash
pkg install curl
curl -fsSL https://clipilot.themobileprof.com/clio | sh
```

The installer places `clio` and `clio-run-module` in `$PREFIX/bin/`.

## Why `clio-run-module` exists

Android blocks some syscalls inside Go binaries. **Module workflows always run via bash:**

```bash
clio-run-module termux_setup setup
```

Never paste module YAML manually — let Clio download it.

## First-run path

1. Install Clio
2. Run `clio`
3. Complete **`setup termux`** wizard (or type `setup`)
4. `sync` when online
5. Start asking questions

## 32-bit vs 64-bit phones

Older phones need the **arm** binary, not arm64. The install script detects `armv7l` / `armv8l` automatically.

## Storage access

The Termux setup wizard runs `termux-setup-storage` so you can reach Downloads and shared storage at `~/storage/`.

## Pidgin on your phone

```
>> wetin dey inside downloads folder
>> abeg check disk space
```

## Troubleshooting

| Problem | Fix |
|---------|-----|
| `clio-run-module: not found` | Re-run install script |
| Module not found | `sync` or `download <id>` |
| No match offline | Rephrase or enable network for remote search |

## Practice

Run `setup termux` or confirm it was completed, then ask three Pidgin queries.
""",
    },
    {
        "title": "Lesson 7 — Practice & Next Steps",
        "description": "Capstone exercises, command reference, and where to get help.",
        "durationMinutes": 12,
        "content": """# Practice & Next Steps

## Capstone challenges

Complete these without Googling the raw commands:

1. **Files** — "show hidden files with details in this folder"
2. **Space** — "check how much disk space is left"
3. **Search** — "find files larger than 50 megabytes"
4. **Setup** — run one setup wizard you have not tried yet
5. **Module** — browse `modules`, download one, and read its description

## Quick command reference

| Goal | Clio command |
|------|--------------|
| All wizards | `setup` |
| All automation modules | `modules` |
| Both, separated | `catalog` |
| Download one module | `download <id>` |
| Sync from server | `sync` / `sync full` |
| Help | `help` |
| Quit | `exit` |

## When Clio cannot match

- Rephrase more simply: verb + noun ("list files", "check disk")
- Try `catalog` to see what exists
- Check you are online if you expect a remote match
- For setup tasks, try `setup <name>` explicitly

## Keep learning

- **CLIPilot registry** — more modules added over time (`sync` to update)
- **GitHub** — github.com/themobileprof/clio
- **TheMobileProf courses** — combine with Linux and Git micro courses

## You finished when…

- [ ] You can install and launch Clio on Termux or Linux
- [ ] You distinguish setup wizards from automation modules
- [ ] You can download and run at least one module
- [ ] You ask questions in plain English or Pidgin confidently

**Stop Googling. Start asking.**
""",
    },
]


def load_env() -> tuple[str, str, str]:
    env_path = Path(__file__).resolve().parents[1] / ".env"
    if env_path.exists():
        for line in env_path.read_text().splitlines():
            line = line.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            key, _, value = line.partition("=")
            os.environ.setdefault(key.strip(), value.strip())

    base = os.environ.get("LMS_BASE_URL", "").rstrip("/")
    email = os.environ.get("LMS_EMAIL", "")
    password = os.environ.get("LMS_PASSWORD", "")
    if not base or not email or not password:
        print("Missing LMS_BASE_URL, LMS_EMAIL, or LMS_PASSWORD in .env", file=sys.stderr)
        sys.exit(1)
    return base, email, password


def request(method: str, url: str, token: str | None = None, body: dict | None = None) -> dict:
    data = json.dumps(body).encode() if body is not None else None
    headers = {"Content-Type": "application/json", "Accept": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=60) as resp:
            raw = resp.read().decode()
            return json.loads(raw) if raw else {}
    except urllib.error.HTTPError as e:
        detail = e.read().decode()
        raise RuntimeError(f"{method} {url} -> {e.code}: {detail}") from e


def main() -> None:
    base, email, password = load_env()

    auth = request("POST", f"{base}/auth/admin/login", body={"email": email, "password": password})
    token = auth["token"]
    print(f"✓ Logged in as {auth['user']['email']}")

    upsert = request("POST", f"{base}/admin/micro-courses/upsert", token, COURSE)
    course = upsert["microCourse"]
    course_id = course["id"]
    created = upsert.get("created", False)
    print(f"✓ Micro course {'created' if created else 'updated'}: {course['title']} ({course_id})")

    existing_lessons = request("GET", f"{base}/admin/micro-courses/{course_id}", token)
    lesson_count = len(existing_lessons.get("microCourse", {}).get("lessons", []) or [])
    lessons = load_lessons()
    if lesson_count >= len(lessons):
        print(f"✓ Course already has {lesson_count} lessons — skipping lesson creation")
        return

    for i, lesson in enumerate(lessons, 1):
        result = request(
            "POST",
            f"{base}/admin/courses/{course_id}/lessons",
            token,
            {
                "title": lesson["title"],
                "description": lesson["description"],
                "content": lesson["content"],
                "durationMinutes": lesson["durationMinutes"],
            },
        )
        lid = result.get("lesson", {}).get("id", "?")
        print(f"  ✓ Lesson {i}/{len(lessons)}: {lesson['title']} ({lid})")

    print("\n✅ Clio micro course published successfully.")
    print(f"   Title: {COURSE['title']}")
    print(f"   Lessons: {len(lessons)}")
    print(f"   Duration: {COURSE['duration']}")


if __name__ == "__main__":
    main()
