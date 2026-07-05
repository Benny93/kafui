# Consumer groups & offsets — group list with lag, then the group detail
Output vhs/gifs/consumer-groups.gif
Source vhs/_config.tape

Hide
Type "./kafui --mock --resource consumer-groups" Enter
Sleep 3s
Show

Sleep 2500ms
Down@400ms 3
Sleep 1s
# Open the consumer-group detail (topic-grouped partition lag)
Enter
Sleep 3s
Escape
Sleep 1500ms

Hide
Ctrl+C
