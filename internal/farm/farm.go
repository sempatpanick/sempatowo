package farm

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
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

// Bot automates OwO farming for one Discord account.
type Bot struct {
	token  string
	client *discord.Client
	log    *util.Logger
	// env is the process environment, read and validated once at startup.
	env *config.Env

	// cfg is installed by onReady on the gateway goroutine while message
	// handlers, timers, and the queue goroutine read it concurrently, so the
	// pointer itself is stored atomically. config.Loader guards its own
	// contents; this only protects the handoff.
	cfg atomic.Pointer[config.Loader]

	mu             sync.Mutex
	active         bool // false during captcha
	ready          bool
	captchaSolving bool
	huntbotStarted bool
	huntbot        *huntbot.Handler
	gamble         *gamble.Manager
	daily          *daily.Manager
	autoQuest      *quest.Manager

	// sender owns the outgoing message queue and its own lock.
	sender *sender
	// sleeper owns the huntbot upgrader's cancellable wait and its own lock.
	sleeper sleeper

	// sched owns the farm command heap and its own lock.
	sched farmSchedState

	checklistAwaiting bool

	timerCancel map[string]func()
	questOwo    *questProgress

	// stats guards everything learned from OwO's replies with its own lock.
	stats *farmStats

	// captchaTimers owns the countdown warnings and its own lock.
	captchaTimers captchaTimers

	simulateCaptcha bool
}

type questProgress struct {
	current, total int
	cancel         func()
	waitCh         chan struct{}
}

// New creates a bot instance (does not connect yet).
func New(token string, env *config.Env) *Bot {
	if env == nil {
		env = &config.Env{}
	}
	b := &Bot{
		token:       token,
		env:         env,
		log:         util.NewLogger(""),
		active:      true,
		timerCancel: make(map[string]func()),
		stats:       newFarmStats(),
	}
	b.sender = newSender(
		b.canSend,
		func() time.Duration { return b.settings().SendMessageInterval.Std() },
		func(channel, text string) { b.send(channel, text, false) },
		func(name string, fn func()) { util.Go(b.logDanger, name, fn) },
	)
	return b
}

// Connection supervision. The gateway library resumes on its own, but only for
// a bounded number of attempts (gateway.MaxReconnectAttempts), and a single
// failed re-dial ends the read loop for good — after that nothing is running
// and the process would otherwise sit idle looking healthy. Run rebuilds the
// client when that happens.
const (
	// connectionPoll is how often the supervisor samples gateway state.
	connectionPoll = 15 * time.Second
	// connectionGrace must exceed the library's own backoff ladder
	// (1+2+4+8+16s) so we never fight its resume with a full reconnect.
	connectionGrace = 90 * time.Second

	reconnectBackoffMin = 5 * time.Second
	reconnectBackoffMax = 5 * time.Minute
)

// Run connects to Discord and supervises the session, rebuilding it with
// backoff whenever the gateway drops for good. It returns only when the
// connection cannot be established at all.
func (b *Bot) Run() error {
	backoff := reconnectBackoffMin
	for attempt := 1; ; attempt++ {
		err := b.run()
		if err != nil {
			b.logDanger("Connection failed: " + err.Error())
			if attempt == 1 {
				// Never connected — likely a bad token or no network, and
				// retrying forever would just hide that.
				return err
			}
		} else {
			// The session was live before it dropped, so start the ladder over
			// rather than inheriting the delay from an earlier outage.
			b.logDanger("Gateway connection lost")
			backoff = reconnectBackoffMin
		}

		b.teardownSession()

		b.logInfo(fmt.Sprintf("Reconnecting in %s", backoff))
		time.Sleep(backoff)
		if backoff < reconnectBackoffMax {
			backoff *= 2
			if backoff > reconnectBackoffMax {
				backoff = reconnectBackoffMax
			}
		}
	}
}

// teardownSession stops every background worker and clears per-session state so
// a fresh client starts clean. Mirrors the pause performed by handleCaptcha.
func (b *Bot) teardownSession() {
	b.mu.Lock()
	b.ready = false
	b.captchaSolving = false
	b.stopFarmTimersLocked()
	b.clearCaptchaTimersLocked()
	b.active = true
	b.mu.Unlock()

	// The sender and the scheduler each own a lock. Both are stopped outside
	// b.mu: b.mu may be held while taking theirs, so the reverse must never
	// happen.
	b.sched.Stop()
	b.sender.Clear()
	b.sender.Stop()

	b.stopHuntbot()
	b.stopGamble()
	b.stopDaily()
	b.stopAutoQuest()

	if c := b.discordClient(); c != nil {
		_ = c.Close()
	}
}

// superviseConnection blocks until the gateway has been down long enough that
// the library has clearly given up resuming.
func (b *Bot) superviseConnection(client *discord.Client) {
	if client == nil || client.Gateway == nil {
		return
	}
	superviseLoop(client.Gateway.IsConnected, connectionPoll, connectionGrace,
		func() { b.logDanger("Gateway disconnected — waiting for resume") },
		func() { b.logInfo("Gateway reconnected") })
}

// superviseLoop returns once connected has reported false continuously for at
// least grace. Split from superviseConnection so the timing logic is testable
// without a live gateway.
func superviseLoop(connected func() bool, poll, grace time.Duration, onDown, onUp func()) {
	ticker := time.NewTicker(poll)
	defer ticker.Stop()

	var downSince time.Time
	for range ticker.C {
		if connected() {
			if !downSince.IsZero() {
				downSince = time.Time{}
				if onUp != nil {
					onUp()
				}
			}
			continue
		}

		if downSince.IsZero() {
			downSince = time.Now()
			if onDown != nil {
				onDown()
			}
			continue
		}

		if time.Since(downSince) >= grace {
			return
		}
	}
}

func (b *Bot) run() error {
	client, err := discord.New(b.token)
	if err != nil {
		return err
	}
	b.client = client

	// These run on the library's goroutines and parse untrusted OwO text, so
	// each one recovers rather than letting a bad message kill the process
	// (and with it every other account sharing it).
	client.OnReady(func() {
		defer util.Recover(b.logDanger, "onReady")
		b.onReady()
	})

	client.OnMessageCreate(func(msg *discord.Message) {
		defer util.Recover(b.logDanger, "onMessage")
		b.onMessage(msg)
	})

	client.OnMessageUpdate(func(msg *discord.Message) {
		defer util.Recover(b.logDanger, "onMessageUpdate")
		b.onMessageUpdate(msg)
	})

	client.OnRawEvent(func(event string, data json.RawMessage) {
		defer util.Recover(b.logDanger, "onRawGateway")
		b.onRawGateway(event, data)
	})

	if err := client.Open(); err != nil {
		return err
	}

	b.superviseConnection(client)
	return nil
}

func (b *Bot) onReady() {
	user := b.discordUser()
	if user == nil {
		b.logDanger("Ready event fired but user is nil")
		return
	}
	b.log.SetID(user.Username)

	// onReady fires again after every reconnect. NewLoader starts a file
	// watcher goroutine that cannot be stopped, so reuse the existing loader
	// rather than leaking one per reconnect — same account, same config file.
	loader := b.cfg.Load()
	if loader != nil {
		b.onSessionReady()
		return
	}

	loader, res, err := config.NewLoader(b.env.Dirs.Config, user.ID.String(), user.Username, b.log, b.onConfigChange)
	if err != nil {
		b.log.Danger("Config error: " + err.Error())
		return
	}
	b.cfg.Store(loader)

	if res.Created {
		b.log.Info("Created " + loader.Path() + " from defaults")
	}
	for _, note := range res.Notes {
		b.log.Info("config: " + note)
	}

	s := loader.Get()
	for _, w := range s.Warnings() {
		b.log.Info("config warning: " + w)
	}
	b.log.Info(fmt.Sprintf("Channels — farm: %s, quest: %s", s.FarmChannel(), s.QuestChannel()))

	b.onSessionReady()
}

// onConfigChange restarts only the subsystems whose settings actually moved.
// The loader hands over both versions precisely so this can be selective: a
// blanket restart tore down the farm scheduler and every timer on any edit,
// which reset in-flight delays and lost the gamble martingale state for an
// unrelated one-character change.
func (b *Bot) onConfigChange(old, new config.Settings) {
	b.log.Info("Config reloaded")

	// Anything routed through the message queue depends on these.
	global := old.Prefix != new.Prefix ||
		old.DefaultChannel != new.DefaultChannel ||
		old.OwoBotID != new.OwoBotID ||
		old.SendMessageInterval != new.SendMessageInterval

	if global || farmSchedulingChanged(old, new) {
		b.restartFarmScheduler()
	}
	if global || old.Features.Huntbot != new.Features.Huntbot {
		b.restartHuntbot()
	}
	if global || old.Features.Gamble != new.Features.Gamble || old.TrackBalance != new.TrackBalance {
		b.restartGamble()
	}
	if global || old.Features.Daily != new.Features.Daily || old.TrackBalance != new.TrackBalance {
		b.restartDaily()
	}
	if global || old.Features.Quest != new.Features.Quest {
		b.restartAutoQuest()
	}
}

// farmSchedulingChanged reports whether any timer-driven command moved.
func farmSchedulingChanged(old, new config.Settings) bool {
	o, n := old.Features, new.Features
	return o.Hunt != n.Hunt ||
		o.Battle != n.Battle ||
		o.Pray != n.Pray ||
		o.Curse != n.Curse ||
		o.Zoo != n.Zoo ||
		o.Inventory != n.Inventory ||
		o.Checklist != n.Checklist ||
		o.Quest != n.Quest
}

// onSessionReady starts automation for a freshly established session. Split out
// of onReady so reconnects can reuse it without rebuilding the config loader.
func (b *Bot) onSessionReady() {
	if b.simulateCaptcha {
		b.scheduleSimulateCaptcha()
		return
	}

	time.AfterFunc(8*time.Second, func() {
		defer util.Recover(b.logDanger, "startupDelay")
		// Test and set under a single lock. Discord re-dispatches READY on
		// resume, and two callbacks racing across separate critical sections
		// could both pass the check and start automation twice.
		b.mu.Lock()
		start := !b.ready && b.canSendLocked()
		if start {
			b.ready = true
		}
		b.mu.Unlock()
		if !start {
			return
		}
		b.startAutomation()
	})
}

func (b *Bot) settings() config.Settings {
	loader := b.cfg.Load()
	if loader == nil {
		return config.Defaults()
	}
	return loader.Get()
}

func (b *Bot) onMessage(msg *discord.Message) {
	if msg == nil {
		return
	}
	s := b.settings()
	if msg.Author == nil || msg.Author.ID.String() != s.OwoBotID {
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
	if msg.ChannelID.String() == s.FarmChannel() && !b.captchaSolving {
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
	if ch != s.FarmChannel() && msg.GuildID != 0 {
		return
	}
	// MESSAGE_UPDATE often omits author; OwO edits in the hunt channel are still ours.
	if msg.Author != nil && msg.Author.ID.String() != s.OwoBotID {
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
		gm.AuthorID = s.OwoBotID
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
	b.stopFarmTimersLocked()
	b.mu.Unlock()

	b.sched.Stop()
	b.sender.Clear()
	b.sender.Stop()

	b.stopHuntbot()
	b.stopGamble()
	b.stopDaily()
	b.stopAutoQuest()

	util.Go(b.logDanger, "captchaFlow", b.runCaptchaFlow)
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
	notify.CaptchaUrgent(profile, "Solve captcha now! Auto-solver also running in background.", url, b.env.Notifications)
	b.scheduleCaptchaWarnings()
	captcha.OpenBrowserAsync(url, profile, b.env.Browser, stillNeeded)

	if !b.env.Captcha.AutoSolveEnabled() {
		b.logDanger("No CAPTCHA_API_KEY — solve manually in the browser tab that just opened")
		return
	}

	b.logInfo("Auto-solver running in background (browser already opened as fallback)...")
	util.Go(b.logDanger, "captchaSolve", func() {
		result := captcha.Solve(b.token, b.env.Captcha)
		if !stillNeeded() {
			return
		}
		if result.Success {
			b.logInfo("OwO captcha solved automatically — " + result.Message)
			return
		}
		b.logDanger("Auto-solve failed — use the browser tab: " + result.Message)
	})
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
			defer util.Recover(b.logDanger, "captchaWarning")
			b.mu.Lock()
			solving := b.captchaSolving
			b.mu.Unlock()
			if !solving {
				return
			}
			url := captcha.GetURL(b.token)
			msg := fmt.Sprintf("%d minute(s) left to solve captcha or you may be banned", minLeft)
			b.logDanger(msg)
			notify.CaptchaUrgent(profile, msg, url, b.env.Notifications)
			// stillNeeded := func() bool {
			// 	b.mu.Lock()
			// 	defer b.mu.Unlock()
			// 	return b.captchaSolving
			// }
			// captcha.OpenBrowserAsync(url, profile, stillNeeded)
		})
		b.captchaTimers.Add(t)
	}
	t := time.AfterFunc(captchaDeadline, func() {
		defer util.Recover(b.logDanger, "captchaDeadline")
		captcha.ReleaseBrowserSlot(profile, b.env.Browser.Isolated)
		b.logDanger("Captcha deadline reached — account may be banned. Solve manually if the page is still open.")
	})
	b.captchaTimers.Add(t)
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
		captcha.ReleaseBrowserSlot(b.profileLabel(), b.env.Browser.Isolated)
	}

	b.logInfo("OwO verification success — resuming auto farm (" + content + ")")
	b.startAutomation()
}

func (b *Bot) clearCaptchaTimers() {
	b.mu.Lock()
	wasSolving := b.clearCaptchaTimersLocked()
	b.mu.Unlock()
	if wasSolving {
		captcha.ReleaseBrowserSlot(b.profileLabel(), b.env.Browser.Isolated)
	}
}

// clearCaptchaTimersLocked requires b.mu, and reports whether a captcha was
// actually outstanding — only then does the browser slot need releasing.
func (b *Bot) clearCaptchaTimersLocked() bool {
	b.captchaTimers.Clear()
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
	cl := checklistState{
		daily:   !strings.Contains(desc, "⬛ 🎁"),
		vote:    !strings.Contains(desc, "⬛ 📝"),
		quest:   !strings.Contains(desc, "⬛ 📜"),
		lootbox: !strings.Contains(desc, "⬛ 💎"),
		crate:   !strings.Contains(desc, "⬛ ⚔"),
	}

	if strings.Contains(desc, "⬛ 🍪") && s.Features.Cookie.Enabled {
		target := s.Features.Cookie.Target
		if target == "" {
			target = s.OwoBotID
		}
		b.enqueue(s.FarmChannel(), b.randomPrefix([]string{"cookie"})+" <@"+target+">")
		cl.cookie = true
	}
	b.stats.SetChecklist(cl)

	b.log.Info(fmt.Sprintf("Checklist: daily=%v cookie=%v quest=%v lootbox=%v crate=%v",
		cl.daily, cl.cookie, cl.quest, cl.lootbox, cl.crate))

	if cl.allDone() {
		b.log.Info("All checklist completed")
		if s.StopWhenChecklistDone {
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

// [\d,]+ because OwO comma-groups anything from 1,000 up; \d+ dropped every
// four-digit gain, silently under-counting totalXP.
var xpRe = regexp.MustCompile(`gained \*\*([\d,]+)xp\*\*`)
var questRe = regexp.MustCompile(`(?s)\*\*\d+\. (.+?)\*\*.*?Progress: \[(\d+)/(\d+)\]`)

func (b *Bot) handleHuntGems(content, nick string) {
	if !isHuntMessage(content, nick) {
		return
	}

	b.stats.AddHunt()
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

	if len(missing) > 0 && s.Features.Gems.Enabled {
		inv := b.stats.Inventory()
		var gemIDs []string
		for _, gemType := range missing {
			if id := bestGem(inv, items.Gems[gemType]); id != "" {
				gemIDs = append(gemIDs, id)
			}
		}
		if len(gemIDs) > 0 {
			b.useGems(gemIDs)
		}
	}

	for _, m := range xpRe.FindAllStringSubmatch(content, -1) {
		// Atoi alone would fail on "2,800" and silently add zero.
		if xp, ok := util.ParseAmount(m[1]); ok {
			b.stats.AddXP(xp)
		}
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
	b.stats.SetInventory(inv)
	s := b.settings()

	if inv[items.Crate] > 0 && s.Features.Crate.Enabled {
		b.enqueue(s.FarmChannel(), b.randomPrefix([]string{"crate"})+" all")
	}
	if inv[items.Lootbox] > 0 && s.Features.Lootbox.Enabled {
		b.enqueue(s.FarmChannel(), b.randomPrefix([]string{"lootbox", "lb"})+" all")
	}
	if inv[items.LootboxFabled] > 0 && s.Features.Lootbox.Fabled {
		b.enqueue(s.FarmChannel(), b.randomPrefix([]string{"lootbox", "lb"})+" fabled all")
	}
}

func (b *Bot) handleQuest(msg *discord.Message, nick string) {
	s := b.settings()
	if s.AutoQuestActive() {
		return
	}
	if !s.Features.Quest.Enabled {
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
		delay := s.Features.Quest.OwoDelay
		channel := s.FarmChannel()
		b.mu.Unlock()
		b.runOwoQuest(done, delay, channel)
	}
}

func (b *Bot) runOwoQuest(done <-chan struct{}, spacing config.Range, channel string) {
	if spacing.IsZero() {
		spacing = config.Defaults().Features.Quest.OwoDelay
	}
	go func() {
		defer util.Recover(b.logDanger, "owoQuestLoop")
		for {
			// Re-drawn each pass so the messages are not evenly spaced.
			delay := time.NewTimer(spacing.Pick())
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
	b.stats.ConsumeItems(ids)
	s := b.settings()
	b.enqueue(s.FarmChannel(), b.randomPrefix([]string{"use"})+" "+strings.Join(ids, " "))
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
	// No-op unless status.checklist is enabled.
	b.scheduleNextChecklist()
}

func (b *Bot) startFarmSchedulerIfNeeded() {
	if b.farmSchedulerNeeded(b.settings()) {
		b.startFarmScheduler()
	}
}

func (b *Bot) restartFarmScheduler() {
	b.mu.Lock()
	ready := b.ready && b.canSendLocked()
	b.mu.Unlock()
	if !ready {
		return
	}
	if b.farmSchedulerNeeded(b.settings()) {
		b.startFarmScheduler()
	} else if b.sched.Running() {
		b.sched.Stop()
	}
}

func (b *Bot) farmSchedulerNeeded(s config.Settings) bool {
	return s.Features.Hunt.Enabled || s.Features.Battle.Enabled || s.Features.Pray.Enabled ||
		s.Features.Curse.Enabled || s.Features.Zoo.Enabled || s.Features.Inventory.Enabled || s.Features.Quest.Enabled
}

func (b *Bot) stopFarmTimers() {
	b.sched.Stop()
	b.mu.Lock()
	b.stopFarmTimersLocked()
	b.mu.Unlock()
}

// stopFarmTimersLocked requires b.mu. It deliberately does not touch the
// scheduler: that has its own lock and is stopped by the caller beforehand.
func (b *Bot) stopFarmTimersLocked() {
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
		defer util.Recover(b.logDanger, "timer:"+name)
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
	b.enqueue(b.settings().FarmChannel(), b.randomPrefix([]string{"cl", "checklist"}))
}

// actionDelay returns the next delay in milliseconds for a scheduled command.
// An inverted range is swapped rather than rejected: Validate already refuses
// to load one, so reaching here means a default was somehow bypassed and
// farming through it beats stopping.
func (b *Bot) actionDelay(kind string) int {
	s := b.settings()
	var r config.Range
	switch kind {
	case "hunt":
		r = s.Features.Hunt.Delay
	case "battle":
		r = s.Features.Battle.Delay
	}
	if r.IsZero() {
		r = config.Defaults().Features.Hunt.Delay
	}
	if r.Max < r.Min {
		r.Min, r.Max = r.Max, r.Min
	}
	return int(r.Pick() / time.Millisecond)
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

func (b *Bot) enqueue(channel, text string) { b.sender.Send(channel, text) }

func (b *Bot) enqueueGambleBet(channel, text, game string) { b.sender.SendBet(channel, text, game) }

func (b *Bot) signalGambleResult(game string) { b.sender.SignalGambleResult(game) }

func (b *Bot) stopQueue() { b.sender.Stop() }

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

// --- helpers ---

func contextWithCancel() (done <-chan struct{}, cancel func()) {
	ch := make(chan struct{})
	return ch, func() { close(ch) }
}
