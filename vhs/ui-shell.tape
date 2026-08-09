# UI shell — help overlay, theme switching, and the resource picker
Output vhs/gifs/ui-shell.gif
Source vhs/_config.tape

Hide
Type "./kafui --mock" Enter
Sleep 3s
Show

Sleep 1500ms
# Full help overlay
Type "?"
Sleep 2500ms
Type "?"
Sleep 800ms
# Cycle the theme from the command palette — it no longer squats on `T`.
Type ":"
Sleep 800ms
Type "theme"
Sleep 1000ms
Enter
Sleep 1800ms
Type ":"
Sleep 800ms
Type "theme"
Sleep 1000ms
Enter
Sleep 1800ms
# The command palette also reaches every resource list.
Type ":"
Sleep 500ms
Type "brokers"
Sleep 800ms
Enter
Sleep 2s
# The contextual actions menu lists everything you can do to the selection,
# with each entry's key and mnemonic beside it.
Type "a"
Sleep 2500ms
Escape
Sleep 1s

Hide
Ctrl+C
