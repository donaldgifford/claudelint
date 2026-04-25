# Companion scaffolding

This directory holds scaffolding for sibling repos. Nothing here is
compiled into the `claudelint` binary, and nothing here ships with a
release — the files are checked in so that reviewers can see them in
the same PR as the `claudelint` changes that anticipate them.

## `claudelint-action/`

The composite GitHub Action specified in DESIGN-0002 §3 and
IMPL-0002 Phase 2.9. Meant to live at `donaldgifford/claudelint-action`
as a separate public repo.

To bootstrap the companion repo:

```bash
# From this directory:
tmp=$(mktemp -d)
cp -R claudelint-action "${tmp}/"
cd "${tmp}/claudelint-action"
git init -q
git add -A
git commit -m "feat: initial claudelint-action v1"
gh repo create donaldgifford/claudelint-action \
    --public \
    --description "GitHub Action for claudelint" \
    --license apache-2.0 \
    --source=. --push
git tag v1.0.0
git tag v1
git push origin v1.0.0 v1
```

After the companion repo is pushed and the `v1` floating tag is in
place, this directory can be deleted from claudelint on the next
Phase 3 commit — it only exists to make review easier.
