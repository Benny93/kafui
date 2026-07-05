# Message browsing — live consumption with metadata, headers, and decoded payloads
Output vhs/gifs/messages.gif
Source vhs/_config.tape

Hide
Type "./kafui --mock --resource topics" Enter
Sleep 3s
Show

# Pick an active topic and open it the way a user would (full partition data).
Sleep 1500ms
Down@300ms 6
Sleep 800ms
Enter
Sleep 1s
# Ensure live tailing is running, then watch messages stream in.
Space
Sleep 5s
Down@350ms 4
Sleep 1s
# Open a message's detail view (key/value/headers/metadata).
Enter
Sleep 3s
Escape
Sleep 1500ms

Hide
Ctrl+C
