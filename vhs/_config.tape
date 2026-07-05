# Shared VHS settings for all kafui feature demos.
# Sourced by each feature tape right after its Output line.
Require kafui

# A wide terminal + smaller font so every resource table's fixed columns fit
# the content pane (the widest, Connectors, needs ~140 cols). See opentodos.md
# item #1 — the real fix is auto-fitting columns; this just makes the demos
# render without the enclosing box wrapping over-wide rows.
Set Shell "bash"
Set FontSize 15
Set Width 1840
Set Height 900
Set Padding 12
Set Margin 12
Set MarginFill "#1A1A2E"
Set BorderRadius 8
Set TypingSpeed 55ms

# Env must come AFTER all Set commands — VHS ignores any Set that follows a
# non-setting command. Isolate the demo from (and protect) the real
# ~/.config/kafui/config.yaml: kafui reads/writes its config under $HOME, and
# the theme toggle in some demos persists it. Point HOME at a throwaway dir
# seeded with known-good defaults (sidebar on, dark theme, release-check off) so
# every GIF is deterministic and the user's real config is never touched.
# render-all.sh seeds this dir.
Env HOME "/tmp/kafui-vhs-home"
