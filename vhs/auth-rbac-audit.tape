# Auth, RBAC & audit — read-only mode: header badge + a denied mutation.
# (RBAC profiles + the effective-permissions "whoami" view appear once a
#  permission profile is configured; by default kafui runs unrestricted, so the
#  visible surface is read-only enforcement + the audit log.)
Output vhs/gifs/auth-rbac-audit.gif
Source vhs/_config.tape

Hide
Type "./kafui --mock --read-only --resource topics"
Enter
Sleep 3s
Show

# The header carries the "read-only" badge; every mutation is denied + audited.
Sleep 3s
Down@400ms 5
Sleep 600ms
Enter
Sleep 2s
# Attempt to produce — denied because the cluster is read-only.
Type "P"
Sleep 3s

Hide
Ctrl+C
