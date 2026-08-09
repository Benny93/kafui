# Command palette — `:` reaches every destination and every application
# command, by typing part of its name. It replaces the per-screen jump keys the
# app used to have (C, K, ctrl+g, ctrl+t, ctrl+w).
Output vhs/gifs/command-palette.gif
Source vhs/_config.tape

Hide
Type "./kafui --mock --resource topics" Enter
Sleep 4s
Show

Sleep 1500ms
# Open it. Everything you can reach is listed, with its direct key where it has
# one — which is how the shortcuts get learned.
Type ":"
Sleep 2500ms

# Type to search. Matching is a substring of the entry name, so a few letters
# are enough.
Type "brok"
Sleep 2s
Enter
Sleep 2500ms

# Every screen is reachable the same way, including the ones with no sidebar
# entry at all.
Type ":"
Sleep 1200ms
Type "metr"
Sleep 1800ms
Enter
Sleep 2500ms

# It is not only navigation: application commands live here too.
Type ":"
Sleep 1200ms
Type "theme"
Sleep 1800ms
Enter
Sleep 2000ms
Type ":"
Sleep 1000ms
Type "theme"
Sleep 1200ms
Enter
Sleep 2000ms

# Arrow keys work as well as typing, and esc dismisses without running anything.
Type ":"
Sleep 1200ms
Down@350ms 4
Sleep 1500ms
Escape
Sleep 1500ms

# Back to a resource list, again by name.
Type ":"
Sleep 1000ms
Type "consumer"
Sleep 1800ms
Enter
Sleep 2500ms

Hide
Ctrl+C
