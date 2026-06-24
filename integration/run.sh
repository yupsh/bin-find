#!/bin/sh
# Integration checks for yup-find, run inside a Debian (GNU coreutils +
# findutils) container.
#
# parity ARGS...  — yup-find ARGS must match GNU `find` ARGS, AFTER both outputs
#                   are piped through `sort`. find's traversal order is
#                   filesystem-dependent, so ordering differences are not real
#                   divergences; sorting both sides removes that noise.
# assert WANT ARGS... — yup-find ARGS must produce WANT exactly (used where
#                   yup-find diverges from GNU by design; see cmd-find
#                   COMPATIBILITY.md).
set -eu

fails=0

# A known directory tree, so parity is reproducible regardless of the host FS.
#   /work
#   ├── a.txt
#   ├── b.log
#   └── sub
#       ├── c.txt
#       └── deep
#           └── d.log
root=/work
mkdir -p "$root/sub/deep"
touch "$root/a.txt" "$root/b.log" "$root/sub/c.txt" "$root/sub/deep/d.log"

parity() {
	ours=$(yup-find "$@" 2>/dev/null | sort || true)
	gnu=$(find "$@" 2>/dev/null | sort || true)
	if [ "$ours" = "$gnu" ]; then
		printf 'ok    parity  find %s\n' "$*"
	else
		printf 'FAIL  parity  find %s\n        gnu:  %s\n        ours: %s\n' "$*" "$gnu" "$ours"
		fails=$((fails + 1))
	fi
}

assert() {
	want=$1
	shift
	got=$(yup-find "$@" 2>/dev/null | sort || true)
	if [ "$got" = "$want" ]; then
		printf 'ok    assert  find %s\n' "$*"
	else
		printf 'FAIL  assert  find %s\n        want: %s\n        got:  %s\n' "$*" "$want" "$got"
		fails=$((fails + 1))
	fi
}

# Full-tree recursion from a root path.
parity "$root"

# -type f / -type d. Both GNU find and yup-find accept the `-type f` argument
# form (cli/v3 accepts the single-dash spelling of a long flag), so the argument
# vectors are byte-identical on both sides.
parity "$root" -type f
parity "$root" -type d

# -maxdepth: descend at most N levels below the root (root is depth 0).
parity "$root" -maxdepth 0
parity "$root" -maxdepth 1
parity "$root" -maxdepth 2

# -type and -maxdepth combined.
parity "$root" -type f -maxdepth 2
parity "$root" -type d -maxdepth 1

# Documented divergence: a `.`-rooted walk. GNU find prefixes every descendant
# with `./` (`./a.txt`); yup-find walks via afero, which joins from the bare root
# and emits `a.txt`. The root entry itself (`.`) is identical. We assert the
# yup-find form rather than parity here.
cd "$root"
assert "$(printf '.\na.txt\nb.log\nsub\nsub/c.txt\nsub/deep\nsub/deep/d.log')"

# Same divergence with an explicit `.` operand.
assert "$(printf '.\na.txt\nb.log\nsub\nsub/c.txt\nsub/deep\nsub/deep/d.log')" .

# -type still parity-matches under a `.` root once both are sorted, because the
# leading `./` is the only difference and -type f yields the same *set* shape;
# but to avoid the prefix noise we assert the yup-find form for files only.
assert "$(printf 'a.txt\nb.log\nsub/c.txt\nsub/deep/d.log')" . -type f

if [ "$fails" -ne 0 ]; then
	printf '\n%s check(s) failed\n' "$fails"
	exit 1
fi
printf '\nall checks passed\n'
