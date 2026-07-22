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
	b.mu.Lock()
	if b.farmAwaiting == nil {
		b.farmAwaiting = make(map[string]struct{})
	}
	b.farmAwaiting[name] = struct{}{}
	b.mu.Unlock()

	go func() {
		defer util.Recover(b.logDanger, "farmResponseTimeout:"+name)
		time.Sleep(farmResponseTimeout)
		b.mu.Lock()
		_, pending := b.farmAwaiting[name]
		if pending {
			delete(b.farmAwaiting, name)
		}
		can := pending && b.canSendLocked() && b.cmdSchedStop != nil
		b.mu.Unlock()
		if !can {
			return
		}
		def := b.farmCmdByName(name)
		if def != nil && def.enabled(b) {
			b.rescheduleFarmCmd(name)
		}
	}()
}

func (b *Bot) signalFarmResponse(name string) {
	b.mu.Lock()
	if b.farmAwaiting == nil {
		b.mu.Unlock()
		return
	}
	if _, ok := b.farmAwaiting[name]; !ok {
		b.mu.Unlock()
		return
	}
	delete(b.farmAwaiting, name)
	can := b.canSendLocked() && b.cmdSchedStop != nil
	b.mu.Unlock()
	if !can {
		return
	}
	def := b.farmCmdByName(name)
	if def == nil || !def.enabled(b) {
		return
	}
	b.rescheduleFarmCmd(name)
}

func (b *Bot) rescheduleFarmCmd(name string) {
	def := b.farmCmdByName(name)
	if def == nil {
		return
	}
	delay := def.delayMs(b)
	if delay <= 0 {
		return
	}
	b.mu.Lock()
	if b.canSendLocked() && b.cmdSchedStop != nil && def.enabled(b) {
		b.pushScheduledCmd(name, time.Now().Add(time.Duration(delay)*time.Millisecond))
	}
	b.mu.Unlock()
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
	if interval <= 0 {
		return
	}
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
