---
name: supervision-labels
description: Use when deriving offline ten-minute session-write labels from Pair supervision fact JSONL segments.
---

# Supervision labels

Run from the repository root with Python 3.9 or newer; no packages are required:

```sh
python3 scripts/supervision-labels/labels.py /path/to/supervision.jsonl /path/to/older-segment.jsonl
```

Select local segments covering the observation window plus ten minutes afterward.
Input order does not matter; identical generation/sequence duplicates are removed.
Output is JSONL on stdout, one row per retained non-generated hub delivery, with
the original opaque identities, hint and delivery outcome plus `label` and `reason`.
The script creates no files and changes no Couch behavior.

Interpret labels as observations:

- `true`: a same-subject write in the same supervisor generation has a later
  sequence and elapsed time in `(delivery, delivery + 600000 ms]`.
- `false`: contiguous sequence evidence covers that horizon without a write.
  This means no observed write, not neglect or a failed delivery.
- `null`: insufficient evidence; preserve the reason when analyzing results.

Delivery outcome remains confirmed, uncertain or cancelled independently of the
label. Never treat a wording hint as intent. Slot hashes cannot be joined across
supervisor generations. Incarnations are recorded but the label joins the subject.

Malformed or unsupported rows are counted on stderr without echoing their content
and disable all negative labels in the batch. Conflicting duplicates are excluded
as evidence; conflicts, backwards clocks and reported drops disable negatives for
their generation. Valid positive observations remain usable. Missing sequences
and truncated horizons produce unknown labels.

The reader refuses batches exceeding 64 MiB, 100,000 input rows or 32 KiB per
line with nonzero exit and no labels. Select fewer/shorter segments and retain
enough horizon coverage; do not remove damaged rows to manufacture negatives.

Verify changes with:

```sh
python3 -m unittest discover -s scripts/supervision-labels -v
```
