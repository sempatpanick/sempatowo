package farm

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	discord "github.com/hytams/discordgo-self"
	"github.com/semptpanick/sempatowo/internal/captcha"
	"github.com/semptpanick/sempatowo/internal/config"
	"github.com/semptpanick/sempatowo/internal/daily"
	"github.com/semptpanick/sempatowo/internal/gamble"
	"github.com/semptpanick/sempatowo/internal/huntbot"
	"github.com/semptpanick/sempatowo/internal/items"
	"github.com/semptpanick/sempatowo/internal/notify"
	"github.com/semptpanick/sempatowo/internal/quest"
	"github.com/semptpanick/sempatowo/internal/util"
)

const captchaDeadline = 10 * time.Minute

const gambleResultWait = 8 * time.Second

type queuedMsg struct {
	channel    string
	text       string
	waitGamble string // "coinflip" or "slots" — pause queue until result
}

// Bot automates OwO farming for one Discord account.
type Bot struct {
	token  string
	client *discord.Client
	log    *util.Logger
	cfg    *config.Loader

	mu                  sync.Mutex
	active              bool // false during captcha
	ready               bool
	captchaSolving        bool
	huntbotStarted        bool
	huntbot             *huntbot.Handler
	gamble              *gamble.Manager
	daily               *daily.Manager
	autoQuest           *quest.Manager
	sleepMu               sync.Mutex
	sleep                 *sleepHandle

	queue       []queuedMsg
	queueStop   chan struct{}
	queueRunning bool

	gambleWaitMu   sync.Mutex
	gambleWaitCh   chan struct{}
	gambleWaitGame string

	cmdHeap      cmdHeap
	cmdSeq       uint64
	cmdSchedStop chan struct{}
	farmAwaiting map[string]struct{}

	checklistAwaiting bool

	timerCancel map[string]func()
	inventory   map[string]int
	checklist   checklistState
	totalXP     int
	totalHunts  int
	lastBattleLog string
	questOwo    *questProgress

	captchaTimers []*time.Timer

	simulateCaptcha bool
}

type checklistState struct {
	daily, vote, cookie, quest, lootbox, crate bool
}

type questProgress struct {
	current, total int
	cancel         func()
	waitCh         chan struct{}
}

// New creates a bot instance (does not connect yet).
func New(token string) *Bot {
	return &Bot{
		token:       token,
		log:         util.NewLogger(""),
		active:      true,
		timerCancel: make(map[string]func()),
		inventory:   make(map[string]int),
	}
}

// Run connects to Discord and blocks until disconnect.
func (b *Bot) Run() error {
	return b.run()
}

func (b *Bot) run() error {
	client, err := discord.New(b.token)
	if err != nil {
		return err
	}
	b.client = client

	client.OnReady(func() {
		b.onReady()
	})

	client.OnMessageCreate(func(msg *discord.Message) {
		b.onMessage(msg)
	})

	client.OnMessageUpdate(func(msg *discord.Message) {
		b.onMessageUpdate(msg)
	})

	client.OnRawEvent(func(event string, data json.RawMessage) {
		b.onRawGateway(event, data)
	})

	if err := client.Open(); err != nil {
		return err
	}

	select {} // run until killed
}

func (b *Bot) onReady() {
	user := b.discordUser()
	if user == nil {
		b.logDanger("Ready event fired but user is nil")
		return
	}
	b.log.SetID(user.Username)

	loader, err := config.NewLoader("config", user.Username, func(s config.Settings) {
		b.log.Info("Config reloaded")
		b.restartFarmScheduler()
		b.restartHuntbot()
		b.restartGamble()
		b.restartDaily()
		b.restartAutoQuest()
	})
	if err != nil {
		b.log.Danger("Config error: " + err.Error())
		return
	}
	b.cfg = loader

	s := b.cfg.Get()
	b.log.Info(fmt.Sprintf("Channels — hunt: %s, quest: %s", s.Channels.Hunt, s.Channels.Quest))

	if b.simulateCaptcha {
		b.scheduleSimulateCaptcha()
		return
	}

	b.enqueue(s.Channels.Hunt, "sempatowo v1.0.0")
	// b.startChecklistLoop()

	time.AfterFunc(8*time.Second, func() {
		b.mu.Lock()
		ready := !b.ready && b.canSendLocked()
		b.mu.Unlock()
		if !ready {
			return
		}
		b.mu.Lock()
		b.ready = true
		b.mu.Unlock()
		b.startAutomation()
	})
}

func (b *Bot) settings() config.Settings {
	if b.cfg == nil {
		return config.Defaults()
	}
	return b.cfg.Get()
}

func (b *Bot) onMessage(msg *discord.Message) {
	if msg == nil {
		return
	}
	s := b.settings()
	if msg.Author == nil || msg.Author.ID.String() != s.OwoID {
		return
	}

	content := normalizeZW(msg.Content)

	// Verification can arrive without a mention — handle before isForMe.
	if isVerificationSuccess(content) {
		b.handleVerificationSuccess(msg.Content)
		return
	}

	// Captcha must be handled before isForMe / gamble — stops farm immediately.
	if b.tryHandleCaptcha(msg, content) {
		return
	}

	nick := b.nickname(msg)
	if msg.ChannelID.String() == s.Channels.Hunt && !b.captchaSolving {
		b.handleGambleMessage(msg)
	}

	if !b.isForMe(msg, nick) {
		return
	}

	b.handleDailyMessage(msg, nick)
	b.handleAutoQuestMessage(msg, nick)

	if b.captchaSolving {
		return
	}

	b.logOwOResponse(msg, content, nick)
	b.trySignalFarmFromMessage(msg, content, nick)
	b.handleChecklist(msg, nick)
	b.handleHuntGems(msg.Content, nick)
	b.handleInventory(msg.Content, nick)
	b.handleQuest(msg, nick)
	b.handleHuntbotMessage(msg)
}

func (b *Bot) onMessageUpdate(msg *discord.Message) {
	if msg == nil || b.captchaSolving {
		return
	}
	s := b.settings()
	ch := msg.ChannelID.String()
	if ch != s.Channels.Hunt && msg.GuildID != 0 {
		return
	}
	// MESSAGE_UPDATE often omits author; OwO edits in the hunt channel are still ours.
	if msg.Author != nil && msg.Author.ID.String() != s.OwoID {
		return
	}

	content := normalizeZW(msg.Content)
	if b.tryHandleCaptcha(msg, content) {
		return
	}

	if b.gamble == nil {
		return
	}

	gm := b.toGambleMessage(msg)
	gm.Content = content
	if gm.AuthorID == "" {
		gm.AuthorID = s.OwoID
	}
	b.gamble.HandleMessageUpdate(gm)

	if b.captchaSolving {
		return
	}
	nick := b.nickname(msg)
	if !b.isForMe(msg, nick) {
		return
	}
	if shouldSkipOwOLog(content) {
		return
	}
	b.logOwOResponse(msg, content, nick)
	b.trySignalFarmFromMessage(msg, content, nick)
	b.handleAutoQuestMessage(msg, nick)
}

func (b *Bot) handleGambleMessage(msg *discord.Message) {
	if b.gamble == nil {
		return
	}
	nick := b.nickname(msg)
	gm := b.toGambleMessage(msg)
	gm.Content = normalizeZW(msg.Content)
	b.gamble.HandleCash(gm.Content, nick)
	b.gamble.HandleMessage(gm)
	b.gamble.HandleGambleResult(gm)
}

func (b *Bot) nickname(msg *discord.Message) string {
	if msg == nil {
		return b.username()
	}
	if msg.Member != nil && msg.Member.Nick != "" {
		return msg.Member.Nick
	}
	client := b.discordClient()
	user := b.discordUser()
	if client != nil && client.State != nil && user != nil && msg.GuildID != 0 {
		if member, ok := client.State.GetMember(msg.GuildID, user.ID); ok && member != nil && member.Nick != "" {
			return member.Nick
		}
	}
	return b.username()
}

func (b *Bot) isForMe(msg *discord.Message, nick string) bool {
	if msg == nil {
		return false
	}
	uid := b.userID()
	if uid == "" {
		return false
	}
	if msg.GuildID == 0 {
		return true
	}
	for _, u := range msg.Mentions {
		if u != nil && u.ID.String() == uid {
			return true
		}
	}
	if strings.Contains(msg.Content, uid) {
		return true
	}
	if msg.Interaction != nil && msg.Interaction.User != nil && msg.Interaction.User.ID.String() == uid {
		return true
	}
	uname := b.username()
	for _, name := range []string{nick, uname} {
		if name == "" {
			continue
		}
		if strings.Contains(msg.Content, name) {
			return true
		}
		for _, embed := range msg.Embeds {
			if embedContainsSafe(embed, name) {
				return true
			}
		}
	}
	return false
}

func normalizeZW(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= 0x200B && r <= 0x200D || r == 0xFEFF {
			return -1
		}
		return r
	}, s)
}

func isVerificationSuccess(content string) bool {
	lower := strings.ToLower(content)
	return strings.Contains(lower, "verified that you are human") ||
		strings.Contains(lower, "thank you!") ||
		strings.Contains(content, "Thank you! :3") ||
		strings.Contains(content, "👍")
}

func (b *Bot) canSendLocked() bool {
	return b.active && !b.captchaSolving
}

func (b *Bot) canSend() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.canSendLocked()
}

func (b *Bot) handleCaptcha() {
	b.mu.Lock()
	if b.captchaSolving {
		b.mu.Unlock()
		return
	}
	b.captchaSolving = true
	b.active = false
	b.ready = false
	b.queue = nil
	b.stopQueueLocked()
	b.stopFarmTimersLocked()
	b.mu.Unlock()
	b.stopHuntbot()
	b.stopGamble()
	b.stopDaily()
	b.stopAutoQuest()

	go b.runCaptchaFlow()
}

func (b *Bot) runCaptchaFlow() {
	b.logDanger("OwO captcha detected — ~10 minutes to solve or account may be banned")

	profile := b.username()
	stillNeeded := func() bool {
		b.mu.Lock()
		defer b.mu.Unlock()
		return b.captchaSolving
	}

	url := captcha.GetURL(b.token)
	notify.CaptchaUrgent(profile, "Solve captcha now! Auto-solver also running in background.", url)
	b.scheduleCaptchaWarnings()
	captcha.OpenBrowserAsync(url, profile, stillNeeded)

	if strings.TrimSpace(getEnv("CAPTCHA_API_KEY")) == "" {
		b.logDanger("No CAPTCHA_API_KEY — solve manually in the browser tab that just opened")
		return
	}

	b.logInfo("Auto-solver running in background (browser already opened as fallback)...")
	go func() {
		result := captcha.Solve(b.token)
		if !stillNeeded() {
			return
		}
		if result.Success {
			b.logInfo("OwO captcha solved automatically — " + result.Message)
			return
		}
		b.logDanger("Auto-solve failed — use the browser tab: " + result.Message)
	}()
}

func (b *Bot) profileLabel() string {
	if u := b.username(); u != "" {
		return u
	}
	if b.token != "" && len(b.token) > 12 {
		return b.token[:12]
	}
	return "account"
}

func (b *Bot) scheduleCaptchaWarnings() {
	warnAt := []int{8, 5, 2, 1}
	profile := b.profileLabel()
	for _, minLeft := range warnAt {
		delay := captchaDeadline - time.Duration(minLeft)*time.Minute
		minLeft := minLeft
		t := time.AfterFunc(delay, func() {
			b.mu.Lock()
			solving := b.captchaSolving
			b.mu.Unlock()
			if !solving {
				return
			}
			url := captcha.GetURL(b.token)
			msg := fmt.Sprintf("%d minute(s) left to solve captcha or you may be banned", minLeft)
			b.logDanger(msg)
			notify.CaptchaUrgent(profile, msg, url)
			// stillNeeded := func() bool {
			// 	b.mu.Lock()
			// 	defer b.mu.Unlock()
			// 	return b.captchaSolving
			// }
			// captcha.OpenBrowserAsync(url, profile, stillNeeded)
		})
		b.mu.Lock()
		b.captchaTimers = append(b.captchaTimers, t)
		b.mu.Unlock()
	}
	t := time.AfterFunc(captchaDeadline, func() {
		captcha.ReleaseBrowserSlot(profile)
		b.logDanger("Captcha deadline reached — account may be banned. Solve manually if the page is still open.")
	})
	b.mu.Lock()
	b.captchaTimers = append(b.captchaTimers, t)
	b.mu.Unlock()
}

func (b *Bot) handleVerificationSuccess(content string) {
	b.mu.Lock()
	if !b.captchaSolving {
		b.mu.Unlock()
		return
	}
	wasSolving := b.clearCaptchaTimersLocked()
	b.active = true
	b.ready = true
	b.mu.Unlock()

	if wasSolving {
		captcha.ReleaseBrowserSlot(b.profileLabel())
	}

	b.logInfo("OwO verification success — resuming auto farm (" + content + ")")
	// b.startChecklistLoop()
	b.startAutomation()
}

func (b *Bot) clearCaptchaTimers() {
	b.mu.Lock()
	wasSolving := b.clearCaptchaTimersLocked()
	b.mu.Unlock()
	if wasSolving {
		captcha.ReleaseBrowserSlot(b.profileLabel())
	}
}

func (b *Bot) clearCaptchaTimersLocked() bool {
	for _, t := range b.captchaTimers {
		t.Stop()
	}
	b.captchaTimers = nil
	wasSolving := b.captchaSolving
	b.captchaSolving = false
	return wasSolving
}

func (b *Bot) handleChecklist(msg *discord.Message, nick string) {
	embed := firstEmbed(msg)
	if embed == nil || embed.Author == nil {
		return
	}
	if !strings.Contains(embed.Author.Name, nick+"'s Checklist") {
		return
	}
	desc := embed.Description
	s := b.settings()

	b.log.Info("Checking checklist")
	b.checklist.daily = !strings.Contains(desc, "⬛ 🎁")

	if strings.Contains(desc, "⬛ 🍪") && s.Status.Cookie {
		target := s.Target.Cookie
		if target == "" {
			target = s.OwoID
		}
		b.enqueue(s.Channels.Hunt, b.randomPrefix([]string{"cookie"})+" <@"+target+">")
		b.checklist.cookie = true
	}

	b.checklist.vote = !strings.Contains(desc, "⬛ 📝")
	b.checklist.quest = !strings.Contains(desc, "⬛ 📜")
	b.checklist.lootbox = !strings.Contains(desc, "⬛ 💎")
	b.checklist.crate = !strings.Contains(desc, "⬛ ⚔")

	cl := b.checklist
	b.log.Info(fmt.Sprintf("Checklist: daily=%v cookie=%v quest=%v lootbox=%v crate=%v",
		cl.daily, cl.cookie, cl.quest, cl.lootbox, cl.crate))

	if cl.daily && cl.cookie && cl.quest && cl.lootbox && cl.crate {
		b.log.Info("All checklist completed")
		if s.ChecklistCompleted {
			b.stopFarmTimers()
			b.mu.Lock()
			if !b.ready && b.canSendLocked() {
				b.ready = true
				b.mu.Unlock()
				b.startAutomation()
				return
			}
			b.mu.Unlock()
			return
		}
	}

	b.mu.Lock()
	if !b.ready && b.canSendLocked() {
		b.ready = true
		b.mu.Unlock()
		b.signalChecklistResponse()
		b.startAutomation()
		return
	}
	b.mu.Unlock()
	b.signalChecklistResponse()
}

var huntGemRe = regexp.MustCompile(`(?:(.+)\*\*( spent|, hunt))`)
var inventoryRe = regexp.MustCompile("`(\\d+|2--)`<a?:\\w+:\\d+>([⁰¹²³⁴⁵⁶⁷⁸⁹]+)")
var xpRe = regexp.MustCompile(`gained \*\*(\d+)xp\*\*`)
var questRe = regexp.MustCompile(`(?s)\*\*\d+\. (.+?)\*\*.*?Progress: \[(\d+)/(\d+)\]`)

func (b *Bot) handleHuntGems(content, nick string) {
	if !isHuntMessage(content, nick) {
		return
	}

	b.totalHunts++
	s := b.settings()
	var missing []string
	if !strings.Contains(content, "gem1") {
		missing = append(missing, "huntgem")
	}
	if !strings.Contains(content, "gem3") {
		missing = append(missing, "empgem")
	}
	if !strings.Contains(content, "gem4") {
		missing = append(missing, "luckgem")
	}

	if len(missing) > 0 && s.Status.Gems {
		var gemIDs []string
		for _, gemType := range missing {
			if id := bestGem(b.inventory, items.Gems[gemType]); id != "" {
				gemIDs = append(gemIDs, id)
			}
		}
		if len(gemIDs) > 0 {
			b.useGems(gemIDs)
		}
	}

	for _, m := range xpRe.FindAllStringSubmatch(content, -1) {
		xp, _ := strconv.Atoi(m[1])
		b.totalXP += xp
	}
}

func isHuntMessage(content, nick string) bool {
	if strings.Contains(content, "You found:") {
		return true
	}
	if strings.Contains(content, "hunt is empowered") || strings.Contains(content, ", hunt") {
		return true
	}
	if strings.Contains(content, nick+"** spent") {
		return true
	}
	return huntGemRe.MatchString(content)
}

func bestGem(inv map[string]int, ids []string) string {
	for _, id := range ids {
		if inv[id] > 0 {
			return id
		}
	}
	return ""
}

func (b *Bot) handleInventory(content, nick string) {
	if !strings.Contains(content, nick+"'s Inventory") {
		return
	}
	inv := make(map[string]int)
	for _, m := range inventoryRe.FindAllStringSubmatch(content, -1) {
		inv[m[1]] = util.SuperscriptToNumber(m[2])
	}
	b.inventory = inv
	s := b.settings()

	if inv[items.Crate] > 0 && s.Status.Crate {
		b.enqueue(s.Channels.Hunt, b.randomPrefix([]string{"crate"})+" all")
	}
	if inv[items.Lootbox] > 0 && s.Status.Lootbox {
		b.enqueue(s.Channels.Hunt, b.randomPrefix([]string{"lootbox", "lb"})+" all")
	}
	if inv[items.LootboxFabled] > 0 && s.Status.LootboxFabled {
		b.enqueue(s.Channels.Hunt, b.randomPrefix([]string{"lootbox", "lb"})+" fabled all")
	}
}

func (b *Bot) handleQuest(msg *discord.Message, nick string) {
	s := b.settings()
	if s.AutoQuest.Enabled && s.AllowAutoQuest {
		return
	}
	if !s.Status.Quest {
		return
	}
	embed := firstEmbed(msg)
	if embed == nil || embed.Author == nil {
		return
	}
	if !strings.Contains(embed.Author.Name, nick+"'s Quest Log") {
		return
	}

	for _, m := range questRe.FindAllStringSubmatch(embed.Description, -1) {
		if !strings.Contains(m[1], "Say 'owo'") {
			continue
		}
		b.mu.Lock()
		if b.questOwo != nil {
			b.mu.Unlock()
			continue
		}
		current, _ := strconv.Atoi(m[2])
		total, _ := strconv.Atoi(m[3])
		done, cancel := contextWithCancel()
		b.questOwo = &questProgress{current: current, total: total, cancel: cancel}
		intervalMs := s.Interval.Quest.Owo
		channel := s.Channels.Hunt
		b.mu.Unlock()
		b.runOwoQuest(done, intervalMs, channel)
	}
}

func (b *Bot) runOwoQuest(done <-chan struct{}, intervalMs int, channel string) {
	if intervalMs <= 0 {
		intervalMs = 32000
	}
	go func() {
		interval := time.Duration(intervalMs) * time.Millisecond
		for {
			delay := time.NewTimer(interval)
			select {
			case <-done:
				delay.Stop()
				return
			case <-delay.C:
			}

			if !b.canSend() {
				return
			}
			b.mu.Lock()
			q := b.questOwo
			if q == nil {
				b.mu.Unlock()
				return
			}
			waitCh := make(chan struct{})
			q.waitCh = waitCh
			b.mu.Unlock()

			b.logInfo("Owo quest: owo")
			b.enqueue(channel, "owo")

			responseTimer := time.NewTimer(farmResponseTimeout)
			select {
			case <-done:
				responseTimer.Stop()
				return
			case <-waitCh:
				responseTimer.Stop()
			case <-responseTimer.C:
			}

			b.mu.Lock()
			if b.questOwo == nil {
				b.mu.Unlock()
				return
			}
			b.questOwo.current++
			doneQuest := b.questOwo.current >= b.questOwo.total
			if doneQuest {
				b.questOwo = nil
			}
			b.mu.Unlock()
			if doneQuest {
				return
			}
		}
	}()
}

func (b *Bot) useGems(ids []string) {
	for _, id := range ids {
		b.inventory[id]--
		if b.inventory[id] <= 0 {
			delete(b.inventory, id)
		}
	}
	s := b.settings()
	b.enqueue(s.Channels.Hunt, b.randomPrefix([]string{"use"})+" "+strings.Join(ids, " "))
}

// --- Farm scheduler (hunt, battle, pray, etc.) ---

func (b *Bot) startAutomation() {
	if !b.canSend() {
		return
	}
	b.startFarmSchedulerIfNeeded()
	b.startHuntbotIfNeeded()
	b.startGambleIfNeeded()
	b.startDailyIfNeeded()
	b.startAutoQuestIfNeeded()
}

func (b *Bot) startFarmSchedulerIfNeeded() {
	if b.farmSchedulerNeeded(b.settings()) {
		b.startFarmScheduler()
	}
}

func (b *Bot) restartFarmScheduler() {
	b.mu.Lock()
	ready := b.ready && b.canSendLocked()
	schedRunning := b.cmdSchedStop != nil
	b.mu.Unlock()
	if !ready {
		return
	}
	if b.farmSchedulerNeeded(b.settings()) {
		b.startFarmScheduler()
	} else if schedRunning {
		b.mu.Lock()
		b.stopFarmSchedulerLocked()
		b.mu.Unlock()
	}
}

func (b *Bot) farmSchedulerNeeded(s config.Settings) bool {
	return s.Status.Hunt || s.Status.Battle || s.Status.Pray ||
		s.Status.Curse || s.Status.Zoo || s.Status.Inventory || s.Status.Quest
}

func (b *Bot) stopFarmTimers() {
	b.mu.Lock()
	b.stopFarmTimersLocked()
	b.mu.Unlock()
}

func (b *Bot) stopFarmTimersLocked() {
	b.stopFarmSchedulerLocked()
	b.checklistAwaiting = false
	if cancel, ok := b.timerCancel["checklist"]; ok {
		cancel()
		delete(b.timerCancel, "checklist")
	}
	if b.questOwo != nil && b.questOwo.cancel != nil {
		b.questOwo.cancel()
		b.questOwo = nil
	}
}

func (b *Bot) scheduleTimer(name string, delayMs int, fn func()) {
	if old, ok := b.timerCancel[name]; ok {
		old()
	}
	done, cancel := contextWithCancel()
	b.timerCancel[name] = cancel

	go func() {
		select {
		case <-time.After(time.Duration(delayMs) * time.Millisecond):
		case <-done:
			return
		}
		b.mu.Lock()
		active := b.canSendLocked()
		b.mu.Unlock()
		if !active {
			return
		}
		fn()
	}()
}

func (b *Bot) startChecklistLoop() {
	b.sendChecklist()
}

func (b *Bot) sendChecklist() {
	if !b.canSend() {
		return
	}
	b.log.Info("Sending checklist")
	b.markChecklistAwaiting()
	b.enqueue(b.settings().Channels.Hunt, b.randomPrefix([]string{"cl", "checklist"}))
}

func (b *Bot) actionDelay(kind string) int {
	s := b.settings()
	def := config.Defaults()
	var d config.ActionDelay
	switch kind {
	case "hunt":
		d = s.Interval.Hunt
		if d.MinDelay == 0 {
			d = def.Interval.Hunt
		}
	case "battle":
		d = s.Interval.Battle
		if d.MinDelay == 0 {
			d = def.Interval.Battle
		}
	}
	min, max := d.MinDelay, d.MaxDelay
	if min > max {
		min, max = max, min
	}
	if max <= min {
		return min
	}
	return min + rand.Intn(max-min+1)
}

func (b *Bot) randomPrefix(commands []string) string {
	if len(commands) == 0 {
		return "owo"
	}
	s := b.settings()
	prefixes := []string{"owo", s.Prefix}
	if s.Prefix == "" {
		prefixes = []string{"owo", "owo"}
	}
	return prefixes[rand.Intn(2)] + " " + commands[rand.Intn(len(commands))]
}

// --- Message queue ---

func (b *Bot) enqueue(channel, text string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.canSendLocked() || channel == "" || text == "" {
		return
	}
	b.queue = append(b.queue, queuedMsg{channel: channel, text: text})
	if b.queueStop == nil {
		b.startQueueLocked()
	}
}

func (b *Bot) enqueueGambleBet(channel, text, game string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.canSendLocked() || channel == "" || text == "" || game == "" {
		return
	}
	b.queue = append(b.queue, queuedMsg{channel: channel, text: text, waitGamble: game})
	if b.queueStop == nil {
		b.startQueueLocked()
	}
}

func (b *Bot) startQueueLocked() {
	// Caller must hold b.mu.
	if b.queueRunning {
		return
	}
	if b.queueStop == nil {
		b.queueStop = make(chan struct{})
	}
	b.queueRunning = true
	stop := b.queueStop
	go b.runQueue(stop)
}

func (b *Bot) runQueue(stop <-chan struct{}) {
	interval := time.Duration(b.settings().Interval.SendMessage) * time.Millisecond
	if interval <= 0 {
		interval = 5 * time.Second
	}

	for {
		b.mu.Lock()
		if len(b.queue) == 0 || !b.canSendLocked() {
			b.queueRunning = false
			if len(b.queue) == 0 {
				b.queueStop = nil
			}
			b.mu.Unlock()
			return
		}
		msg := b.queue[0]
		b.queue = b.queue[1:]
		b.mu.Unlock()

		if !b.canSend() {
			return
		}
		b.send(msg.channel, msg.text, false)

		if msg.waitGamble != "" {
			b.waitGambleResult(msg.waitGamble, gambleResultWait, stop)
		}

		timer := time.NewTimer(interval)
		select {
		case <-stop:
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (b *Bot) waitGambleResult(game string, timeout time.Duration, stop <-chan struct{}) {
	ch := make(chan struct{})
	b.gambleWaitMu.Lock()
	b.gambleWaitCh = ch
	b.gambleWaitGame = game
	b.gambleWaitMu.Unlock()

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-ch:
	case <-timer.C:
	case <-stop:
	}

	b.gambleWaitMu.Lock()
	if b.gambleWaitCh == ch {
		b.gambleWaitCh = nil
		b.gambleWaitGame = ""
	}
	b.gambleWaitMu.Unlock()
}

func (b *Bot) signalGambleResult(game string) {
	b.gambleWaitMu.Lock()
	if b.gambleWaitGame != game {
		b.gambleWaitMu.Unlock()
		return
	}
	ch := b.gambleWaitCh
	b.gambleWaitCh = nil
	b.gambleWaitGame = ""
	b.gambleWaitMu.Unlock()
	if ch != nil {
		close(ch)
	}
}

func (b *Bot) stopQueue() {
	b.mu.Lock()
	b.stopQueueLocked()
	b.mu.Unlock()
}

func (b *Bot) stopQueueLocked() {
	// Caller must hold b.mu.
	if b.queueStop != nil {
		close(b.queueStop)
		b.queueStop = nil
	}
	b.queueRunning = false
}

func (b *Bot) send(channelID, text string, force bool) {
	b.mu.Lock()
	allowed := force || b.canSendLocked()
	typing := b.settings().Typing
	b.mu.Unlock()

	if !allowed {
		return
	}

	client := b.discordClient()
	if client == nil {
		b.logDanger("Discord client not connected")
		return
	}

	chID, err := discord.ParseSnowflake(channelID)
	if err != nil {
		b.logDanger("Invalid channel ID: " + channelID)
		return
	}

	if typing {
		_ = client.TriggerTyping(chID)
	}
	if _, err := client.SendMessage(chID, text); err != nil {
		b.logDanger("Send error: " + err.Error())
	}
}

// --- Huntbot (separate from manual hunt; shares send queue via enqueue) ---

func (b *Bot) startGambleIfNeeded() {
	g := b.settings().Gamble
	if !g.Coinflip.Enabled && !g.Slots.Enabled && !g.Blackjack.Enabled {
		b.stopGamble()
		return
	}
	if b.gamble == nil {
		b.gamble = gamble.NewManager(b.newGambleContext())
	}
	b.gamble.Start()
	if b.settings().CashCheck {
		b.gamble.RequestCash()
	}
}

func (b *Bot) stopGamble() {
	if b.gamble != nil {
		b.gamble.Stop()
	}
}

func (b *Bot) restartGamble() {
	b.stopGamble()
	b.mu.Lock()
	ready := b.ready && b.canSendLocked()
	b.mu.Unlock()
	if !ready {
		return
	}
	b.startGambleIfNeeded()
}

func (b *Bot) startHuntbotIfNeeded() {
	if !b.settings().Huntbot.Enabled {
		b.stopHuntbot()
		return
	}
	b.mu.Lock()
	if b.huntbotStarted {
		b.mu.Unlock()
		return
	}
	b.huntbotStarted = true
	b.mu.Unlock()

	ctx := b.newHuntbotContext()
	b.huntbot = huntbot.NewHandler(ctx, b.token)
	go b.huntbot.Start()
}

func (b *Bot) stopHuntbot() {
	if b.huntbot != nil {
		b.huntbot.Stop()
		b.huntbot = nil
	}
	b.huntbotStarted = false
}

func (b *Bot) restartHuntbot() {
	b.stopHuntbot()
	b.mu.Lock()
	ready := b.ready && b.canSendLocked()
	b.mu.Unlock()
	if ready {
		b.startHuntbotIfNeeded()
	}
}

func (b *Bot) handleHuntbotMessage(msg *discord.Message) {
	if msg == nil || msg.Author == nil || b.huntbot == nil || !b.settings().Huntbot.Enabled {
		return
	}
	hbMsg := huntbot.Message{
		ChannelID: msg.ChannelID.String(),
		AuthorID:  msg.Author.ID.String(),
		Content:   msg.Content,
	}
	for _, e := range msg.Embeds {
		if e == nil {
			continue
		}
		embed := huntbot.MessageEmbed{}
		if e.Author != nil {
			embed.Author = &huntbot.EmbedAuthor{Name: e.Author.Name}
		}
		for _, f := range e.Fields {
			if f == nil {
				continue
			}
			embed.Fields = append(embed.Fields, huntbot.EmbedField{Name: f.Name, Value: f.Value})
		}
		hbMsg.Embeds = append(hbMsg.Embeds, embed)
	}
	for _, a := range msg.Attachments {
		if a == nil {
			continue
		}
		hbMsg.Attachments = append(hbMsg.Attachments, huntbot.Attachment{URL: a.URL})
	}
	b.huntbot.HandleMessage(hbMsg)
}

type huntbotCtx struct {
	bot *Bot
}

// sleepHandle identifies one in-flight sleep so a sleeper only clears its own.
type sleepHandle struct {
	cancel func()
}

func (b *Bot) newHuntbotContext() *huntbotCtx {
	return &huntbotCtx{bot: b}
}

func (c *huntbotCtx) HuntChannelID() string {
	if c == nil || c.bot == nil {
		return ""
	}
	return c.bot.settings().Channels.Hunt
}
func (c *huntbotCtx) OwoBotID() string {
	if c == nil || c.bot == nil {
		return ""
	}
	return c.bot.settings().OwoID
}
func (c *huntbotCtx) Nickname() string {
	if c == nil || c.bot == nil {
		return ""
	}
	client := c.bot.discordClient()
	user := c.bot.discordUser()
	if client != nil && client.State != nil && user != nil {
		for _, guild := range client.State.Guilds {
			if guild == nil {
				continue
			}
			if member, ok := client.State.GetMember(guild.ID, user.ID); ok && member != nil && member.Nick != "" {
				return member.Nick
			}
		}
	}
	return c.bot.username()
}
func (c *huntbotCtx) Settings() config.Huntbot { return c.bot.settings().Huntbot }
func (c *huntbotCtx) RandomPrefix(cmds []string) string { return c.bot.randomPrefix(cmds) }
func (c *huntbotCtx) CanSend() bool {
	if c == nil || c.bot == nil {
		return false
	}
	return c.bot.canSend()
}
func (c *huntbotCtx) SendMessage(channelID, text string) error {
	if c == nil || c.bot == nil {
		return nil
	}
	c.bot.enqueue(channelID, text)
	return nil
}
func (c *huntbotCtx) Log(msg string) { c.bot.logInfo(msg) }

func (c *huntbotCtx) Sleep(seconds float64) {
	if c == nil || c.bot == nil || seconds <= 0 {
		return
	}
	time.Sleep(time.Duration(seconds * float64(time.Second)))
}

// SleepUntil reports false if CancelSleep cut the wait short, so callers can
// tell an elapsed timer from an aborted one.
func (c *huntbotCtx) SleepUntil(seconds, noise float64) bool {
	if c == nil || c.bot == nil {
		return false
	}
	d := seconds
	if noise > 0 {
		d += rand.Float64() * noise
	}
	if d <= 0 {
		return true
	}

	res := make(chan bool, 1)
	var once sync.Once
	timer := time.AfterFunc(time.Duration(d*float64(time.Second)), func() {
		once.Do(func() { res <- true })
	})
	h := &sleepHandle{cancel: func() {
		once.Do(func() {
			timer.Stop()
			res <- false
		})
	}}

	c.bot.setSleep(h)
	elapsed := <-res
	c.bot.clearSleep(h)
	return elapsed
}

func (c *huntbotCtx) CancelSleep() {
	if c == nil || c.bot == nil {
		return
	}
	c.bot.CancelSleep()
}

// setSleep makes h the active sleep, cancelling whichever one it replaces.
func (b *Bot) setSleep(h *sleepHandle) {
	b.sleepMu.Lock()
	prev := b.sleep
	b.sleep = h
	b.sleepMu.Unlock()
	if prev != nil {
		prev.cancel()
	}
}

func (b *Bot) clearSleep(h *sleepHandle) {
	b.sleepMu.Lock()
	if b.sleep == h {
		b.sleep = nil
	}
	b.sleepMu.Unlock()
}

func (b *Bot) CancelSleep() {
	b.sleepMu.Lock()
	h := b.sleep
	b.sleep = nil
	b.sleepMu.Unlock()
	if h != nil {
		h.cancel()
	}
}

// --- helpers ---

func contextWithCancel() (done <-chan struct{}, cancel func()) {
	ch := make(chan struct{})
	return ch, func() { close(ch) }
}

func getEnv(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}
