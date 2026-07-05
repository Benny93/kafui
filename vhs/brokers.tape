# Broker management — broker list with stats, then the broker detail tabs
Output vhs/gifs/brokers.gif
Source vhs/_config.tape

Hide
Type "./kafui --mock --resource brokers" Enter
Sleep 3s
Show

Sleep 2s
Down@450ms 1
Sleep 1s
# Open the broker detail page
Enter
Sleep 2500ms
# Cycle the Log Dirs / Configs / Metrics tabs
Tab
Sleep 2s
Tab
Sleep 2s
Tab
Sleep 1500ms
Escape
Sleep 1200ms

Hide
Ctrl+C
