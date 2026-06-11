#!/usr/bin/env python3
"""Expand Clio micro course lessons with Gemini and push updates to the LMS."""

from __future__ import annotations

import json
import os
import re
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path

# Reuse lesson outlines from publish script
sys.path.insert(0, str(Path(__file__).resolve().parent))
from publish_clio_micro_course import COURSE, LESSONS_OUTLINE, load_env, request  # noqa: E402

GEMINI_MODEL = "gemini-2.5-flash"
OUTPUT_JSON = Path(__file__).resolve().parent / "clio_course_lessons_gemini.json"

CLIO_CONTEXT = """
Clio facts (must stay accurate):
- Offline-first CLI assistant; natural language → shell commands
- Install: curl -fsSL https://clipilot.themobileprof.com/clio | sh
- REPL prompt: >>
- Commands: setup, modules, catalog, download <id>, sync, sync full, help, exit
- Setup wizards [SETUP WIZARD]: termux, vim, git, devtools, database — via setup <name>
- Automation modules [AUTOMATION]: repeatable tasks — via modules / natural language
- Modules download from CLIPilot registry (not bundled in binary); clio-run-module on Termux
- Understands Nigerian Pidgin e.g. "abeg", "wetin dey", "I wan"
- Lite profile on Termux/low RAM; layers: phrase/catalog, man (non-lite), modules, remote
- Confirm before run for shell commands; match headers show SHELL COMMAND / AUTOMATION MODULE / SETUP WIZARD
Audience: Nigerian students, often on Termux/Android phones, beginners.
Tone: encouraging, practical, clear. Use markdown. Include examples, tables, checklists, practice exercises.
Do NOT invent features Clio does not have.
"""


def gemini_expand(api_key: str, lesson: dict, lesson_num: int, total: int) -> str:
    prompt = f"""{CLIO_CONTEXT}

You are writing Lesson {lesson_num} of {total} for the micro course "{COURSE['title']}".

Lesson title: {lesson['title']}
Lesson summary: {lesson['description']}
Target duration: {lesson['durationMinutes']} minutes

Expand the following draft into a rich, student-friendly lesson (800–1400 words).
Keep markdown format. Structure with:
- Opening hook (why this matters on a phone)
- Clear sections with ## headings
- Code blocks for commands (use ```bash or plain >> prompts)
- At least one table or comparison where useful
- "Try it now" practice steps
- Common mistakes / troubleshooting (short)
- End with a checklist

Draft to expand:
---
{lesson['content']}
---

Return ONLY the lesson markdown body. No preamble."""

    url = (
        f"https://generativelanguage.googleapis.com/v1beta/models/"
        f"{GEMINI_MODEL}:generateContent?key={api_key}"
    )
    body = {
        "contents": [{"parts": [{"text": prompt}]}],
        "generationConfig": {
            "temperature": 0.65,
            "maxOutputTokens": 8192,
        },
    }
    req = urllib.request.Request(
        url,
        data=json.dumps(body).encode(),
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=120) as resp:
        data = json.load(resp)

    text = data["candidates"][0]["content"]["parts"][0]["text"]
    text = text.strip()
    # Strip accidental markdown fences wrapping whole lesson
    if text.startswith("```markdown"):
        text = re.sub(r"^```markdown\s*", "", text)
        text = re.sub(r"\s*```$", "", text)
    elif text.startswith("```"):
        text = re.sub(r"^```\s*", "", text)
        text = re.sub(r"\s*```$", "", text)
    return text.strip()


def gemini_expand_course_description(api_key: str) -> dict[str, str]:
    prompt = f"""{CLIO_CONTEXT}

Write improved marketing/educational copy for this micro course (JSON only):
Title: {COURSE['title']}
Current description: {COURSE['description']}
Objectives: {COURSE['objectives']}

Return strict JSON with keys:
- description (2-3 sentences, compelling for Nigerian students)
- objectives (one paragraph, bullet-style with • characters inline)
- syllabus (short bullet list of 7 lesson topics, one line each)

No markdown code fences. JSON only."""

    url = (
        f"https://generativelanguage.googleapis.com/v1beta/models/"
        f"{GEMINI_MODEL}:generateContent?key={api_key}"
    )
    body = {"contents": [{"parts": [{"text": prompt}]}], "generationConfig": {"temperature": 0.5}}
    req = urllib.request.Request(
        url,
        data=json.dumps(body).encode(),
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=90) as resp:
        data = json.load(resp)
    raw = data["candidates"][0]["content"]["parts"][0]["text"].strip()
    raw = re.sub(r"^```json\s*", "", raw)
    raw = re.sub(r"\s*```$", "", raw)
    return json.loads(raw)


def main() -> None:
    base, email, password = load_env()
    api_key = os.environ.get("GEMINI_API_KEY", "")
    if not api_key:
        print("GEMINI_API_KEY missing in .env", file=sys.stderr)
        sys.exit(1)

    auth = request("POST", f"{base}/auth/admin/login", body={"email": email, "password": password})
    token = auth["token"]
    print(f"✓ Logged in as {auth['user']['email']}")

    lookup = request(
        "GET",
        f"{base}/admin/micro-courses/by-title/{urllib.parse.quote(COURSE['title'])}",
        token,
    )
    course_id = lookup["microCourse"]["id"]
    print(f"✓ Course: {COURSE['title']} ({course_id})")

    lessons_resp = request("GET", f"{base}/admin/courses/{course_id}/lessons?limit=20", token)
    remote_lessons = sorted(lessons_resp["lessons"], key=lambda x: x["order_index"])

    if len(remote_lessons) != len(LESSONS_OUTLINE):
        print(
            f"Warning: remote has {len(remote_lessons)} lessons, local outline has {len(LESSONS_OUTLINE)}",
            file=sys.stderr,
        )

    expanded_lessons: list[dict] = []
    total = len(LESSONS_OUTLINE)

    for i, (outline, remote) in enumerate(zip(LESSONS_OUTLINE, remote_lessons), 1):
        print(f"… Gemini expanding lesson {i}/{total}: {outline['title']}")
        try:
            content = gemini_expand(api_key, outline, i, total)
        except urllib.error.HTTPError as e:
            print(f"  Gemini error: {e.read().decode()}", file=sys.stderr)
            raise
        expanded = {
            **outline,
            "content": content,
            "remoteId": remote["id"],
        }
        expanded_lessons.append(expanded)

        request(
            "PUT",
            f"{base}/admin/lessons/{remote['id']}",
            token,
            {
                "title": outline["title"],
                "description": outline["description"],
                "content": content,
                "durationMinutes": outline["durationMinutes"],
            },
        )
        print(f"  ✓ Updated lesson on LMS ({remote['id']})")
        time.sleep(1.5)  # gentle rate limit

    print("… Gemini improving course description")
    course_copy = gemini_expand_course_description(api_key)
    request(
        "PUT",
        f"{base}/admin/micro-courses/{course_id}",
        token,
        {
            "description": course_copy.get("description", COURSE["description"]),
            "objectives": course_copy.get("objectives", COURSE["objectives"]),
            "syllabus": course_copy.get("syllabus"),
        },
    )
    print("✓ Updated course description/objectives/syllabus")

    OUTPUT_JSON.write_text(
        json.dumps({"course": COURSE, "lessons": expanded_lessons}, indent=2, ensure_ascii=False),
        encoding="utf-8",
    )
    print(f"✓ Saved expanded content to {OUTPUT_JSON.name}")
    print("\n✅ Clio micro course fleshed out with Gemini and saved to LMS.")


if __name__ == "__main__":
    main()
