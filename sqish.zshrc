SQISH_ID="$(hostname)_$(date +%s)_$(shuf -i 1-100 -n 1)"
SQISH=sqish
# Add to history.
function sqish-add () { nohup $SQISH --shell_session_id $SQISH_ID add "$1" >& /dev/null }
autoload add-zsh-hook
add-zsh-hook zshaddhistory sqish-add
# Search history.
function sqish-search () {
        fullscreen=$1
        t=$(mktemp)
        if [[ fullscreen -eq 1 ]]; then
          cmd="search"
        else
          cmd="inline"
        fi
        if [[ $BUFFER != "" ]]; then
          $SQISH --shell_session_id $SQISH_ID $cmd --query $BUFFER 2> $t
        else
          $SQISH --shell_session_id $SQISH_ID $cmd 2> $t
        fi
        cmd=$(cat $t)
        LBUFFER+="$cmd"
}
function sqish-search-fullscreen () {
  sqish-search 1
}
function sqish-search-inline () {
  sqish-search 0
}
zle -N sqish-search-fullscreen
bindkey '^z' sqish-search-fullscreen
zle -N sqish-search-inline
bindkey '^i' sqish-search-inline
