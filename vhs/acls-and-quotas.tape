# ACLs & client quotas — ACL bindings list, then switch to client quotas
Output vhs/gifs/acls-and-quotas.gif
Source vhs/_config.tape

Hide
Type "./kafui --mock --resource acls"
Enter
Sleep 3s
Show

Sleep 2500ms
Down@400ms 2
Sleep 1200ms
# Switch to client quotas via the resource picker
Type ":"
Sleep 500ms
Type "quotas"
Sleep 800ms
Enter
Sleep 3s

Hide
Ctrl+C
