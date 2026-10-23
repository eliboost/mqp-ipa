#!/usr/bin/env python3
"""Generate section index.md pages and wire them into the MkDocs nav.

For each folder referenced by mkdocs.yml that lacks an index.md, write a real
file with the section title and a flat list of links to all direct children.
Existing index.md files (e.g. docs/index.md) are left untouched.

Additionally inject `<section_dir>/index.md` as the first list item of each
nav section that doesn't already reference it. Material's `navigation.indexes`
feature only makes a section heading clickable when its index.md appears as
the section's first unlabeled entry.

Run this before techdocs-cli / mkdocs build so the section indexes exist as
real source files and the nav references them. This replaces the runtime
mkdocs `hooks:` block, which TechDocs strips on build because hooks are not
part of the supported config subset.
"""

import os
import re
import sys
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parent
DOCS_DIR = ROOT / "docs"
MKDOCS_FILE = ROOT / "mkdocs.yml"

TITLE_OVERRIDES = {
    "howto": "HowTo",
    "infra": "Infrastructure",
}

NAV_ITEM_RE = re.compile(r"^(\s*)- (.+):\s*$")


def main():
    with MKDOCS_FILE.open() as f:
        config = yaml.safe_load(f)
    nav = config.get("nav")
    if nav is None:
        print("No nav defined in mkdocs.yml; nothing to do.")
        return 0

    written = []
    injections = []
    _walk(nav, parent_name=None, written=written, injections=injections)

    for path in written:
        print(f"generated {path.relative_to(ROOT)}")

    if injections:
        text = MKDOCS_FILE.read_text()
        new_text = _inject_into_text(text, injections)
        if new_text != text:
            MKDOCS_FILE.write_text(new_text)
            for name, ref in injections:
                print(f"injected {ref} into section {name!r}")

    return 0


def _walk(items, parent_name, written, injections):
    section_dir = _infer_dir(items)
    if section_dir and parent_name is not None and not section_dir.startswith("blog"):
        has_ref = _has_index_ref(items, section_dir)
        if not has_ref:
            injections.append((parent_name, f"{section_dir}/index.md"))
        index_path = DOCS_DIR / section_dir / "index.md"
        if not index_path.exists():
            title = _titleize(section_dir.rsplit("/", 1)[-1])
            links = _link_list(items, section_dir)
            content = f"# {title}\n"
            if links:
                content += "\n" + "\n".join(links) + "\n"
            index_path.parent.mkdir(parents=True, exist_ok=True)
            index_path.write_text(content)
            written.append(index_path)

    for entry in items:
        if isinstance(entry, dict):
            for name, value in entry.items():
                if isinstance(value, list):
                    _walk(value, parent_name=name, written=written, injections=injections)


def _inject_into_text(text, injections):
    names = [name for name, _ in injections]
    duplicates = {n for n in names if names.count(n) > 1}
    if duplicates:
        print(
            f"warning: section names appear more than once, skipping injection for: {', '.join(sorted(duplicates))}",
            file=sys.stderr,
        )
    by_name = {name: ref for name, ref in injections if name not in duplicates}
    pending = set(by_name)

    lines = text.splitlines(keepends=True)
    out = []
    in_nav = False

    for i, line in enumerate(lines):
        stripped = line.rstrip("\n")
        if stripped and not stripped[0].isspace() and stripped.endswith(":"):
            in_nav = stripped.rstrip(":") == "nav"

        out.append(line)

        if not in_nav:
            continue

        match = NAV_ITEM_RE.match(stripped)
        if not match:
            continue
        name = match.group(2)
        if name not in pending:
            continue

        child_indent = _next_child_indent(lines, i)
        if child_indent is None:
            continue
        out.append(" " * child_indent + f"- {by_name[name]}\n")
        pending.discard(name)

    if pending:
        print(
            f"warning: could not locate sections in mkdocs.yml: {', '.join(sorted(pending))}",
            file=sys.stderr,
        )
    return "".join(out)


def _next_child_indent(lines, header_idx):
    for j in range(header_idx + 1, len(lines)):
        if lines[j].strip():
            return len(lines[j]) - len(lines[j].lstrip())
    return None


def _infer_dir(items):
    paths = []
    for entry in items:
        if isinstance(entry, str):
            paths.append(entry)
        elif isinstance(entry, dict):
            for v in entry.values():
                if isinstance(v, str):
                    paths.append(v)
                elif isinstance(v, list):
                    nested = _infer_dir(v)
                    if nested:
                        paths.append(nested + "/_")
    if not paths:
        return None
    common = os.path.commonpath(paths) if len(paths) > 1 else paths[0]
    if common.endswith(".md"):
        common = os.path.dirname(common)
    return common or None


def _has_index_ref(items, section_dir):
    target = f"{section_dir}/index.md"
    for entry in items:
        if entry == target:
            return True
        if isinstance(entry, dict):
            for v in entry.values():
                if v == target:
                    return True
    return False


def _link_list(items, section_dir):
    links = []
    for entry in items:
        if isinstance(entry, str):
            title = _titleize(Path(entry).stem)
            rel = os.path.relpath(entry, section_dir)
            links.append(f"- [{title}]({rel})")
        elif isinstance(entry, dict):
            for title, value in entry.items():
                if isinstance(value, str):
                    rel = os.path.relpath(value, section_dir)
                    links.append(f"- [{title}]({rel})")
                elif isinstance(value, list):
                    first = _first_page(value)
                    if first is not None:
                        rel = os.path.relpath(first, section_dir)
                        links.append(f"- [{title}]({rel})")
                    else:
                        links.append(f"- {title}")
    return links


def _first_page(items):
    for entry in items:
        if isinstance(entry, str):
            return entry
        if isinstance(entry, dict):
            for v in entry.values():
                if isinstance(v, str):
                    return v
                if isinstance(v, list):
                    nested = _first_page(v)
                    if nested:
                        return nested
    return None


def _titleize(name):
    if name in TITLE_OVERRIDES:
        return TITLE_OVERRIDES[name]
    return name.replace("-", " ").replace("_", " ").title()


if __name__ == "__main__":
    sys.exit(main())
