# Fixture project

This is a minimal clean fixture used to smoke-test the claudelint-action
on every push. Keep it boring — every diagnostic added here becomes
part of the Action's test signal.

## Purpose

Exercise the happy path: claudelint runs, finds this file, classifies
it as CLAUDE.md, and returns zero errors.
