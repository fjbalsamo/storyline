Output docs/demo.gif

Set FontSize 16
Set Width 1400
Set Height 620
Set Theme "Catppuccin Mocha"
Set TypingSpeed 40ms

Hide
Type "cd $(mktemp -d) && clear" Enter
Show

Type "storyline init --tools claude" Enter
Sleep 2.5s

Hide
Type "clear" Enter
Show

Type `storyline add --as "club player" --want "to book a court for a free slot" --so-that "I don't have to call the club"` Enter
Sleep 1.5s

Type `storyline add --as "admin" --want "to block courts for maintenance" --so-that "nobody books an unusable court"` Enter
Sleep 1.5s

Type "storyline list" Enter
Sleep 2.5s

Type "storyline validate" Enter
Sleep 3s
