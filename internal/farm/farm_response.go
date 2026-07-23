package farm

import (
	"strings"
	"time"

	discord "github.com/hytams/discordgo-self"
	"github.com/semptpanick/sempatowo/internal/util"
)

const farmResponseTimeout = 30 * time.Second

func farmCmdsFromMessage(msg *discord.Message, content, nick string) []string {
	var cmds []string
	if isHuntMessage(content, nick) {
		cmds = append(cmds, "hunt")
	}
	if isBattleFarmResponse(msg, nick) {
		cmds = append(cmds, "battle")
	}
	if prayTextFromMessage(msg, content) != "" {
		cmds = append(cmds, "pray")
	}
	if isCurseMessage(content) {
		cmds = append(cmds, "curse")
	}
	if isZooMessage(content, nick) {
		cmds = append(cmds, "zoo")
	}
	if nick != "" && strings.Contains(content, nick+"'s Inventory") {
		cmds = append(cmds, "inventory")
	}
	if isQuestLogResponse(msg, nick) {
		cmds = append(cmds, "quest")
	}
	return cmds
}

func isCurseMessage(text string) bool {
	lower := strings.ToLower(text)
	return strings.Contains(lower, "curse") &&
		(strings.Contains(lower, "curses") || strings.Contains(lower, "cursed"))
}

func isZooMessage(content, nick string) bool {
	if nick != "" && strings.Contains(content, nick+"'s zoo") {
		return true
	}
	lower := strings.ToLower(content)
	return nick != "" && strings.Contains(lower, "zoo") && strings.Contains(content, nick)
}

func isBattleFarmResponse(msg *discord.Message, nick string) bool {
	if msg == nil || nick == "" {
		return false
	}
	for _, embed := range msg.Embeds {
		if embed == nil || embed.Author == nil {
			continue
		}
		author := embed.Author.Name
		if strings.Contains(author, nick) && strings.Contains(author, "goes into battle") {
			return true
		}
	}
	return battleFooterFromMessage(msg, nick) != ""
}

func isQuestLogResponse(msg *discord.Message, nick string) bool {
	embed := firstEmbed(msg)
	if embed == nil || embed.Author == nil || nick == "" {
		return false
	}
	return strings.Contains(embed.Author.Name, nick+"'s Quest Log")
}

func isOwoCommandResponse(content, nick string) bool {
	if nick == "" || !strings.Contains(content, nick) {
		return false
	}
	if rankRe.MatchString(content) {
		return true
	}
	lower := strings.ToLower(content)
	return strings.Contains(lower, "here are your") || strings.Contains(lower, "cowoncy")
}

func isChecklistResponse(msg *discord.Message, nick string) bool {
	embed := firstEmbed(msg)
	if embed == nil || embed.Author == nil || nick == "" {
		return false
	}
	return strings.Contains(embed.Author.Name, nick+"'s Checklist")
}

func (b *Bot) trySignalFarmFromMessage(msg *discord.Message, content, nick string) {
	for _, name := range farmCmdsFromMessage(msg, content, nick) {
		b.signalFarmResponse(name)
	}
	if isOwoCommandResponse(content, nick) {
		b.signalOwoQuestResponse()
	}
}

func (b *Bot) markFarmAwaiting(name string) {
	b.sched.MarkAwaiting(name)

	go func() {
		defer util.Recover(b.logDanger, "farmResponseTimeout:"+name)
		time.Sleep(farmResponseTimeout)
		// Claiming is what makes this safe against the reply arriving at the
		// same moment: exactly one of the two wins and reschedules.
		if !b.sched.ClaimAwaiting(name) {
			return
		}
		b.rescheduleFarmCmd(name)
	}()
}

func (b *Bot) signalFarmResponse(name string) {
	if !b.sched.ClaimAwaiting(name) {
		return
	}
	b.rescheduleFarmCmd(name)
}

func (b *Bot) rescheduleFarmCmd(name string) {
	def := b.farmCmdByName(name)
	if def == nil || !def.enabled(b) {
		return
	}
	delay := def.delayMs(b)
	if delay <= 0 {
		return
	}
	if !b.canSend() || !b.sched.Running() {
		return
	}
	b.sched.Push(name, time.Now().Add(time.Duration(delay)*time.Millisecond))
}

func (b *Bot) markChecklistAwaiting() {
	b.mu.Lock()
	b.checklistAwaiting = true
	b.mu.Unlock()

	go func() {
		defer util.Recover(b.logDanger, "checklistResponseTimeout")
		time.Sleep(farmResponseTimeout)
		b.mu.Lock()
		if !b.checklistAwaiting {
			b.mu.Unlock()
			return
		}
		b.checklistAwaiting = false
		b.mu.Unlock()
		b.scheduleNextChecklist()
	}()
}

func (b *Bot) signalChecklistResponse() {
	b.mu.Lock()
	if !b.checklistAwaiting {
		b.mu.Unlock()
		return
	}
	b.checklistAwaiting = false
	b.mu.Unlock()
	b.scheduleNextChecklist()
}

func (b *Bot) scheduleNextChecklist() {
	s := b.settings()
	if !s.Features.Checklist.Enabled {
		return
	}
	interval := int(s.Features.Checklist.Delay.Pick() / time.Millisecond)

	b.mu.Lock()
	// The first schedule after a captcha continues the interrupted wait instead
	// of drawing a fresh interval.
	if left := b.checklistResume; left > 0 {
		b.checklistResume = 0
		interval = int(left / time.Millisecond)
	}
	if interval <= 0 {
		b.checklistDue = time.Time{}
		b.mu.Unlock()
		return
	}
	b.checklistDue = time.Now().Add(time.Duration(interval) * time.Millisecond)
	b.mu.Unlock()

	b.scheduleTimer("checklist", interval, b.startChecklistLoop)
}

func (b *Bot) signalOwoQuestResponse() {
	b.mu.Lock()
	q := b.questOwo
	if q == nil || q.waitCh == nil {
		b.mu.Unlock()
		return
	}
	ch := q.waitCh
	q.waitCh = nil
	b.mu.Unlock()
	close(ch)
}
