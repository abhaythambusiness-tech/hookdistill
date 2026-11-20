# Canonical source of the demo. Render with: vhs assets/demo.tape
# (assets/demo.gif in the repo was rendered from the real command output with ffmpeg
# because vhs was not installed when it was made; re-render with vhs when you can.)
Output assets/demo.gif

Set Shell "bash"
Set FontSize 16
Set Width 1080
Set Height 600
Set Theme "GitHub Dark"
Set TypingSpeed 40ms

Type "hookdistill scan --root examples/sessions --claude-md examples/CLAUDE.md"
Enter
Sleep 3s

Type "hookdistill propose --root examples/sessions --claude-md examples/CLAUDE.md --out /tmp/demo/rules.json"
Enter
Sleep 3s

Type "hookdistill replay c-3902ca48 --root examples/sessions --rules /tmp/demo/rules.json"
Enter
Sleep 4s
