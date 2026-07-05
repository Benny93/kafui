# Application configuration — the read-only effective-config view (global key: ctrl+g)
Output vhs/gifs/application-config.gif
Source vhs/_config.tape

Hide
Type "./kafui --mock" Enter
Sleep 3s
Show

Sleep 1500ms
# Open the effective configuration view (redacted secrets, build info, clusters)
Ctrl+G
Sleep 3s
# Scroll through the config document
Down@250ms 8
Sleep 2s
Down@250ms 8
Sleep 2s

Hide
Ctrl+C
