# Cluster management & dashboard — health, status, per-cluster stats (global key: C)
Output vhs/gifs/cluster-management.gif
Source vhs/_config.tape

Hide
Type "./kafui --mock" Enter
Sleep 3s
Show

Sleep 1500ms
# Open the cluster overview dashboard
Type ":"
Sleep 800ms
Type "cluster dashboard"
Sleep 800ms
Enter
Sleep 3s
# Navigate the cluster rows
Down@450ms 2
Sleep 1500ms
# Toggle the offline-only filter
Type "i"
Sleep 1500ms
Type "i"
Sleep 1500ms

Hide
Ctrl+C
