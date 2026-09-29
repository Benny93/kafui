# Row expand — read a message's full content inline, without opening it
Output vhs/gifs/row-expand.gif
Source vhs/_config.tape

Hide
Type "./kafui --mock --resource topics" Enter
Sleep 3s
Show

# Open a CDC topic: its payloads are too wide for the Value column.
Sleep 1200ms
Down@400ms 2
Sleep 600ms
Enter
Sleep 2s

# Expand the highlighted message: the full value, pretty-printed, right under
# its row. The header shows it is taller than the panel and scrolls.
Type "x"
Sleep 3500ms

# The panel follows the cursor.
Down@1200ms 3
Sleep 1500ms
Up@1200ms 2
Sleep 1500ms

# Collapse again.
Type "x"
Sleep 2s

Hide
Ctrl+C
