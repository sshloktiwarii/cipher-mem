# cipher-mem: The Plain-English Guide

## What does this do?

AI assistants suffer from a fundamental problem: **they have no long-term memory.**

When you start a new conversation or restart your editor, the AI forgets everything:
* It forgets the decisions your team made last month.
* It forgets how you fixed that exact same database bug last week.
* It forgets the documentation you wrote down in your personal notes.

`cipher-mem` connects your AI directly to your personal notes folder (such as an Obsidian vault):
* **Watches your notes automatically:** Whenever you add or edit a markdown file in your folder, `cipher-mem` reads the update and indexes it immediately.
* **Searches by meaning, not just keywords:** If you search for "database crash", it can find a note titled "handling postgres connection timeouts" even if the exact words are different, because it understands the concept.
* **Sub-millisecond speed:** It retrieves the most relevant notes in less than 1 millisecond, without making network calls or uploading your documents to the cloud.

---

## What happens if I don't use this?

Without a memory system, you have to manually copy and paste long documentation files into your AI chat every single time you encounter an issue. This wastes time, increases prompt costs, and relies on you remembering where the relevant note was stored.

---

## Frequently Asked Questions

#### Does this upload my private notes to a company's cloud?
No. All indexing and mathematical vector comparisons happen entirely on your computer's processor. Your notes never leave your machine.

#### Do I need to be using Obsidian?
No. Any folder containing standard Markdown (`.md`) files will work.

#### How do I run it?
In the full Cipher system, `cipher appliance` starts `cipher-mem` automatically. To point it at your own folder standalone:
```bash
./cipher-mem --vault /path/to/my/notes --port 8083
```
