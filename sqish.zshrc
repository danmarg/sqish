SQISH_ID="$(hostname)_$(date +%s)_$(shuf -i 1-100 -n 1)"
SQISH=sqish
# Add to history.
function sqish_add () { $SQISH --shell_session_id $SQISH_ID add "$1" }
autoload add-zsh-hook
add-zsh-hook zshaddhistory sqish_add
# Search history.
function sqish_search() {
  t=$(mktemp)
  $SQISH --shell_session_id $SQISH_ID search 2> $t
  cmd=$(cat $t)
  eval "$cmd"
}

