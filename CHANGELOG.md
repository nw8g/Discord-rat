# changelog - latest stuff n hacks

## v0.7.2 - bugfixs

### fixed

*   **screenshot stability:** no more random "failed to get bitmap bits" errors, added retry logic (3 attempts) + proper delays so `GetDIBits` doesnt shit itself
*   **keylogger bugs:** fixed critical ChaCha20 counter issue that was corrupting logs after a few hours. now properly tracks block counter across sessions so decryption actually works lol
*   **some keylogger bugs:** keys were sometimes logging twice cuz `GetAsyncKeyState` was spamming. added state tracking (`keyPressed[256]bool`) so each keypress only logs once
   
### changed

*   **persistence:** ripped out the sketchy `selfInstall()` + `os.Exit(0)` shit that was causing race conditions, now just points persistence methods directly to current exe location. cleaner & less buggy
*   keylogger polling reduced from 40ms to 10ms for better responsiveness
*   screenshot now does double `GetDIBits` call (info first, then data) - more reliable on different systems

### notes

*   this is basically a "make shit actually work" update, no new features just fixing the broken stuff
*   keylogger encryption is now rock solid, tested with 3+ hour sessions
*   next update probably gonna focus on actual new features instead of fixing my own fuckups lol

---

## v0.7.0 - good update!!

### added

*   **KEYLOGGER MODULE!!!!!!!** fully-featured and stealthy
*   new commands `!keylogger <start/stop/status>` for total control and `!keys` / `!keylogs` to dump the captured shit.
*   uses a custom **dependency-free ChaCha20** implementation to encrypt logs on the fly. Each session gets its own unique nonce so the crypto is solid.
*   Logs are stashed stealthily in the Chrome cache dir (`%APPDATA%`) to avoid suspicion.

### changed

*   **MASSIVE CORE REFACTOR:** the bot connection logic was rewritten from the ground up, with aggressive auto-reconnect and exponential backoff to survive shitty internet connections
*   **LOW-LEVEL SCREENSHOTS:** ditched the thirdparty screenshot library, ,ow uses direct **WinAPI (user32/gdi32)** calls, way stealthier and reduces the final binary size (btw the first time you run the command, sometimes it doesnt work, run it again and it will work)

### notes

*   this drop makes the agent way more stable and adds some serious info-gathering capabilities
*   next on the list is probably proper process injection or a custom obfuscator to make AVs cry.

---

## v0.6.0 control update

### added

* wmi persistence method finally added

### changed

* it upgrades persistence and throws in some new methods, imo its better now

---

## v0.5.2 - idk

###

* better app bound decryption

---

## v0.5.0 - the ghost in the machine 👻

### added

*   `!hide` now takes args (`peb`, `hook`, `spoof`, `all`) for surgical stealth
*   `!admin` & `!system` also take args to pick your exploit (`fodhelper`, `pipe`, etc)
*   `!stealth` command to check what evasion shit is running
*   `!help` show existing commands, duh

### changed

*   rewrote the whole fucking stealth module, it's smarter now & less crashy
*   help menu is updated with all the new toys and examples

### notes

*   this is all about operator control, you choose the weapon
*   opsec is still your problem dont be a dumbass
*   i think i should add an obfuscator btw

---

## v0.3.4 - Shell Stability Fix 安定

### fixed

*   annoying ass `context canceled` bug on `!cmd` & `!shell`/`!ps` is finally squashed.
*   commands were basically getting aborted the second they were sent lol.
*   turns out the timeout context was being a little bitch and dying too early. moved it inside the goroutine, so now shell commands actually have time to run.

### changed

*   upped the default command timeout from 30s to 60s, just in case you run some slow-ass shit.

### notes

*   nothing here

---

## v0.3.3 - cockroach drop 

### added

* auto-persist into `%APPDATA%` now lol, more "stealh"
* new startup trick: drops lil `.url` instead of full exe, sneaky af

### changed

* persistence semi recode 
* reg keys & tasks = disguised as basic MS crap
* logic = hella aggressive, keeps reapplying

### notes

* maybe later ill add wmi methods or hijacking, i dont rlly know

---

## v0.3.2 - third drop? idk lol 🔥

### added

* poc for chrome v127+ decryption (app\_bound, super experimental lol)
* now scans all profs, not just "default"
* bookmarks parser, grabs all da bookmarks
* more chromium targets: beta/dev/canary + funky ones like vivaldi

### changed

* full module refactor, faster & cleaner, easier to add future parsers

### notes

* chrome v127+ decryption is mostly a test/poc, may break next builds
* honestly, no clue if this is really the 3rd drop lol, just rollin' with it

