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
# Cycle the theme: auto -> dark -> light
Type "T"
Sleep 1800ms
Type "T"
Sleep 1800ms
Type "T"
Sleep 1500ms
# Resource picker
Type ":"
Sleep 500ms
Type "brokers"
Sleep 800ms
Enter
Sleep 2s

Hide
Ctrl+C
