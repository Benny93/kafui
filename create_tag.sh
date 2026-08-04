#!/bin/bash
#
# create_tag.sh -- create and push an annotated semantic-version git tag.
#
# If no version is given, the next semantic version is derived automatically
# from the highest existing tag (patch bump by default; use --major/--minor to
# bump a different component). Use --dry-run to preview without changing anything.

set -euo pipefail

DRY_RUN=false
BUMP="patch"        # patch | minor | major
MESSAGE=""
POS_VERSION=""
POS_MESSAGE=""

usage() {
    cat <<EOF
Usage: $0 [options] [tag_version [tag_message]]

Create and push an annotated git tag using a semantic version (MAJOR.MINOR.PATCH).

If tag_version is omitted, the next semantic version is derived automatically
from the highest existing tag (incremented per --major/--minor/--patch, patch
by default). If provided, it must be a valid semantic version and higher than
the latest tag.

Options:
  -n, --dry-run          Print what would be done without creating or pushing.
  -m, --message <msg>    Tag message (default: "Release <version>").
      --major            Bump the major version (e.g. v0.1.33 -> v1.0.0).
      --minor            Bump the minor version (e.g. v0.1.33 -> v0.2.0).
      --patch            Bump the patch version (e.g. v0.1.33 -> v0.1.34). [default]
  -h, --help             Show this help and exit.

Examples:
  $0                                Auto-select next patch version (v0.1.33 -> v0.1.34)
  $0 --minor                        Auto-select next minor version (v0.1.33 -> v0.2.0)
  $0 -n                             Dry run: print the tag that would be created
  $0 v0.2.0 "Release v0.2.0"        Create a specific higher version
EOF
}

die() { echo "ERROR: $*" >&2; exit 1; }

# Strip a leading 'v' from a version string.
strip_v() { echo "${1#v}"; }

# Validate that a string is a MAJOR.MINOR.PATCH semantic version (optional leading v).
semver_ok() {
    local v
    v="$(strip_v "$1")"
    [[ "$v" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]
}

# Compare two dotted versions; prints 1 if a>b, 2 if a<b, 0 if equal.
vercmp() {
    awk -v a="$1" -v b="$2" 'BEGIN{
        n=split(a,aa,"."); m=split(b,bb,".")
        k=(n>m?n:m)
        for(i=1;i<=k;i++){ x=(aa[i]+0); y=(bb[i]+0); if(x>y){print 1; exit} if(x<y){print 2; exit} }
        print 0
    }'
}

# Return the highest existing semantic-version tag (empty if none).
find_latest() {
    local highest="" t c
    while IFS= read -r t; do
        semver_ok "$t" || continue
        if [ -z "$highest" ]; then
            highest="$t"
            continue
        fi
        c="$(vercmp "$(strip_v "$t")" "$(strip_v "$highest")")"
        if [ "$c" = "1" ]; then
            highest="$t"
        fi
    done < <(git tag)
    echo "$highest"
}

# Bump a MAJOR.MINOR.PATCH version by the given component and print vX.Y.Z.
bump() {
    local v maj min pat
    v="$(strip_v "$1")"
    IFS='.' read -r maj min pat <<< "$v"
    case "$2" in
        major) maj=$((maj + 1)); min=0; pat=0 ;;
        minor) min=$((min + 1)); pat=0 ;;
        patch) pat=$((pat + 1)) ;;
    esac
    echo "v${maj}.${min}.${pat}"
}

# --- parse command line ---
POS=()
while [ $# -gt 0 ]; do
    case "$1" in
        -n|--dry-run) DRY_RUN=true; shift ;;
        -m|--message) [ $# -ge 2 ] || die "--message requires a value"; MESSAGE="$2"; shift 2 ;;
        --major)      BUMP="major"; shift ;;
        --minor)      BUMP="minor"; shift ;;
        --patch)      BUMP="patch"; shift ;;
        -h|--help)    usage; exit 0 ;;
        --)           shift; POS+=("$@"); break ;;
        -*)           die "Unknown option: $1 (try --help)" ;;
        *)            POS+=("$1"); shift ;;
    esac
done

[ ${#POS[@]} -le 2 ] || die "Too many arguments (expected at most: tag_version tag_message)"
if [ ${#POS[@]} -ge 1 ]; then POS_VERSION="${POS[0]}"; fi
if [ ${#POS[@]} -ge 2 ]; then POS_MESSAGE="${POS[1]}"; fi

if [ -n "$MESSAGE" ] && [ -n "$POS_MESSAGE" ]; then
    die "Tag message given twice (--message and positional tag_message)"
fi

# --- sanity: must be a git repository ---
git rev-parse --is-inside-work-tree >/dev/null 2>&1 || die "Not inside a git repository"

# --- resolve the target version ---
LATEST="$(find_latest)"
LATEST_CORE="$(strip_v "$LATEST")"

if [ -n "$POS_VERSION" ]; then
    # Explicit version: must be valid and higher than the latest tag.
    semver_ok "$POS_VERSION" || die "Invalid semantic version '$POS_VERSION' (expected MAJOR.MINOR.PATCH, e.g. v1.2.3)"
    TAG_VERSION="v$(strip_v "$POS_VERSION")"
    if [ -n "$LATEST_CORE" ]; then
        case "$(vercmp "$(strip_v "$TAG_VERSION")" "$LATEST_CORE")" in
            0|2) die "$TAG_VERSION is not higher than the latest tag 'v$LATEST_CORE'; use --patch/--minor/--major to bump it automatically" ;;
        esac
    fi
else
    # Auto-select the next semantic version from the highest existing tag.
    if [ -z "$LATEST_CORE" ]; then
        # No existing tags: start a new version history.
        if [ "$BUMP" = "major" ]; then
            TAG_VERSION="v1.0.0"
        else
            TAG_VERSION="v0.1.0"
        fi
    else
        TAG_VERSION="$(bump "$LATEST_CORE" "$BUMP")"
    fi
fi


# --- resolve the tag message ---
if [ -n "$POS_MESSAGE" ]; then
    MESSAGE="$POS_MESSAGE"
fi
if [ -z "$MESSAGE" ]; then
    MESSAGE="Release $TAG_VERSION"
fi

# --- refuse to clobber an existing tag ---
if git rev-parse -q --verify "refs/tags/$TAG_VERSION" >/dev/null 2>&1; then
    die "Tag $TAG_VERSION already exists"
fi

# --- show the resolved values ---
echo "Target tag:    $TAG_VERSION"
echo "Tag message:   $MESSAGE"
if [ -n "$LATEST" ]; then
    echo "Latest tag:    v$LATEST_CORE"
else
    echo "Latest tag:    <none>"
fi
echo "Bump:          $BUMP"

if [ "$DRY_RUN" = true ]; then
    echo
    echo "(dry run) No changes made. Would run:"
    echo "  git tag -a \"$TAG_VERSION\" -m \"$MESSAGE\""
    echo "  git push origin \"$TAG_VERSION\""
    exit 0
fi

# --- create and push ---
git tag -a "$TAG_VERSION" -m "$MESSAGE"
git push origin "$TAG_VERSION"

echo
echo "Tag $TAG_VERSION created and pushed successfully."

