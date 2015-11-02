SQISH_ID="$(hostname)_$(date +%s)_$(shuf -i 1-100 -n 1)"
SQISH=sqish
# Add to history.
function sqish-add () { $SQISH --shell_session_id $SQISH_ID add "$1" }
autoload add-zsh-hook
add-zsh-hook zshaddhistory sqish-add
# Search history.
function sqish-search () {
        t=$(mktemp)
        $SQISH --shell_session_id $SQISH_ID search 2> $t
        cmd=$(cat $t)
        LBUFFER+="$cmd"
}
zle -N sqish-search
bindkey '^z' sqish-search
