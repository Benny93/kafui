# Cluster management & dashboard — health, status, per-cluster stats (global key: C)
Output vhs/gifs/cluster-management.gif
Source vhs/_config.tape

Hide
Type "./kafui --mock" Enter
Sleep 3s
Show

Sleep 1500ms
# Open the cluster overview dashboard
Type "C"
Sleep 3s
# Navigate the cluster rows
Down@450ms 2
Sleep 1500ms
# Toggle the offline-only filter
Type "o"
Sleep 1500ms
Type "o"
Sleep 1500ms

Hide
Ctrl+C
