Output demo.gif

Set Shell "bash"
Set FontSize 14
Set Width 900
Set Height 500
Set Theme "Catppuccin Mocha"
Set Padding 20

Type "zonasul search 'banana' --limit 3"
Sleep 500ms
Enter
Sleep 3s

Sleep 1s

Type "zonasul cart add 33277 --qty 3"
Sleep 500ms
Enter
Sleep 3s

Sleep 1s

Type "zonasul cart --json"
Sleep 500ms
Enter
Sleep 3s
