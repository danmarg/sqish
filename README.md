# SQISH
_SQl Interactive Shell History_

Why store your shell history in easy-to-use plain text files when you can store
it in a structured database instead?

# Usage

Note that I haven't figured out what shell hooks are needed to make this work
for Bash. These instructions are presently ZSH-only.

```
$ source ~/squish.zshrc
$ # do some stuff...
$ ^Z
```

^Z (or `squish search`) brings up an interactive search UI. Press return to
execute the command currently highlighted, type to search via substring, or use
the arrow keys to navigate.
