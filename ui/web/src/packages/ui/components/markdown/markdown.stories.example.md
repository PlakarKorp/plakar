# Heading Level 1

This is a **bold** paragraph with some _italic_ and some `inline code`.

Here is a second paragraph that includes a [link to Plakar](https://plakar.io) and a reference to a [relative page](/docs/getting-started).

---

## Heading Level 2

### Heading Level 3

This paragraph includes:

- An unordered list
- With multiple
- Items

And this paragraph follows immediately after.

---

## Ordered List with Nested Items

1. First item
2. Second item with subitems:
   - Subitem A
   - Subitem B
3. Third item

---

## Blockquote and Emphasis

> This is a blockquote.
>
> It has two paragraphs and uses **bold**, _italic_, and `inline code`.

---

## Code Blocks

### Bash

```bash
$ echo "Hello from Plakar"
$ plakar backup /etc
```

### JSON

```json
{
  "name": "plakar",
  "version": "1.0.0",
  "features": ["immutable", "encrypted", "verifiable"]
}
```

---

## Table

| Feature   | Supported |
| --------- | --------- |
| Backup    | ✅ Yes    |
| Restore   | ✅ Yes    |
| UI Viewer | ✅ Yes    |
| Magic     | ❌ No     |

---

## Images

### ✅ Valid image

![Valid image](https://placehold.co/600x400)

### ❌ Invalid image (should show fallback or alt)

![This image does not exist](https://example.com/invalid-image.jpg)

---

## Horizontal Rule

---

## Checkboxes

- [ ] Unchecked item
- [x] Checked item

---

## Raw HTML

<p>This is a raw HTML paragraph. <strong>It should render correctly</strong> if HTML is enabled.</p>

---

## Inline Elements

Here is some `inline code`, **bold text**, _italic text_, ~~strikethrough~~, and a [link](https://example.com).

---

## Q&A (Simulated)

**Q: Does it support all markdown elements?**
A: Yes, this document is designed to include as many features as possible.

**Q: What happens when content is long?**
A: It should scroll or wrap as per your container’s behavior.
