package main

import (
	"fmt"
	"syncra/internal/ui"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func (m model) View() string {
	if m.suspending {
		return ""
	}

	if m.quitting {
		return "Bye!\n"
	}

	var header, subHeader, body, footer string

	header = ui.HeaderStyle.Render("SYNCRA") + "\n"

	var content string
	switch m.state {
	case stateSetupWorkspace:
		subHeader = ui.SubHeaderStyle.Render("setup / workspace") + "\n"
		inner := ui.MutedStyle.Render("Where should Syncra live?") + "\n\n"
		inner += ui.InfoKeyStyle.Render("path") + "\n" + m.textInput.View()
		if m.err != nil {
			inner += "\n\n" + ui.ErrorTextStyle.Render("! "+m.err.Error())
		}
		content = inner
		footer = ui.FooterStyle.Render("enter: next • esc: quit")

	case stateSetupUsername:
		subHeader = ui.SubHeaderStyle.Render("setup / identity") + "\n"
		inner := ui.MutedStyle.Render("Choose your unique handle.") + "\n\n"
		inner += ui.InfoKeyStyle.Render("username") + "\n" + m.textInput.View()
		if m.err != nil {
			inner += "\n\n" + ui.ErrorTextStyle.Render("! "+m.err.Error())
		}
		content = inner
		footer = ui.FooterStyle.Render("enter: check • esc: quit")

	case stateSetupFullName:
		subHeader = ui.SubHeaderStyle.Render("setup / profile") + "\n"
		inner := ui.MutedStyle.Render("Whatever you like to be called.") + "\n\n"
		inner += ui.InfoKeyStyle.Render("name") + "\n" + m.textInput.View()
		if m.err != nil {
			inner += "\n\n" + ui.ErrorTextStyle.Render("! "+m.err.Error())
		}
		content = inner
		footer = ui.FooterStyle.Render("enter: finish • esc: quit")

	case stateSetupProcessing:
		subHeader = ui.SubHeaderStyle.Render("creating identity...") + "\n"
		content = fmt.Sprintf("\n  %s %s\n", m.spinner.View(), ui.MutedStyle.Render("Generating cryptographic keys & registering..."))
		footer = ""

	case stateSuccess:
		subHeader = ui.SubHeaderStyle.Render("ready") + "\n"
		inner := ui.SuccessStyle().Render("Identity verified.") + "\n\n"
		inner += ui.InfoKeyStyle.Render("user") + ui.InfoValueStyle.Render(m.cfg.Username) + "\n"
		content = inner
		footer = ui.FooterStyle.Render("enter: launch")

	case stateMain:
		subHeader = ui.SubHeaderStyle.Render("active / "+m.cfg.Username) + "\n"

		statusLabel := ui.StatusLabelStyle.Copy().
			Foreground(ui.Success).
			Render("● ONLINE")

		// Status Section
		statusContent := fmt.Sprintf("%s\n\n%s %s\n%s %s",
			statusLabel,
			ui.InfoKeyStyle.Render("identity"), ui.InfoValueStyle.Render(m.cfg.FullName),
			ui.InfoKeyStyle.Render("session"), ui.InfoValueStyle.Render(time.Since(m.startTime).Truncate(time.Second).String()),
		)

		// Chats Section
		friendsList := ""
		if len(m.chats) > 0 {
			friendsList = "\n" + ui.SectionTitleStyle.Render(fmt.Sprintf("RECENT CHATS (%d)", len(m.chats))) + "\n"
			for i, friend := range m.chats {
				cursor := "  "
				style := ui.InfoValueStyle
				if i == m.chatSelectionIndex {
					cursor = lipgloss.NewStyle().Foreground(ui.Primary).Render("» ")
					style = ui.SelectedStyle
				}
				friendsList += fmt.Sprintf("%s %s\n", cursor, style.Render(friend))
			}
		} else {
			friendsList = "\n" + ui.MutedStyle.Render("No recent conversations.")
		}

		content = statusContent + "\n" + friendsList
		if m.isLocal {
			footer = ui.FooterStyle.Render("↑/↓: select chat • s: settings • f: find • l: lan peers • q: quit")
		} else {
			footer = ui.FooterStyle.Render("↑/↓: select chat • s: settings • f: find • q: quit")
		}

	case stateLanNetwork:
		subHeader = ui.SubHeaderStyle.Render("network / lan peers") + "\n"

		var lanList string
		if len(m.lanPeers) > 0 {
			lanList = "\n" + ui.SectionTitleStyle.Render(fmt.Sprintf("ACTIVE DEVICES (%d)", len(m.lanPeers))) + "\n"
			for i, peer := range m.lanPeers {
				cursor := "  "
				style := ui.InfoValueStyle
				if i == m.lanSelectionIndex {
					cursor = lipgloss.NewStyle().Foreground(ui.Primary).Render("» ")
					style = ui.SelectedStyle
				}
				lanList += fmt.Sprintf("%s %s %s\n", cursor, style.Render(peer.FullName), ui.MutedStyle.Render("@"+peer.Username))
			}
		} else {
			lanList = "\n  " + ui.MutedStyle.Render("No active peers found on local network.")
		}

		content = lanList
		footer = ui.FooterStyle.Render("↑/↓: select • enter: chat • esc: back")

	case stateSearch:
		subHeader = ui.SubHeaderStyle.Render("find") + "\n"
		inner := ui.InfoKeyStyle.Render("query") + "\n" + m.searchInput.View() + "\n\n"

		if len(m.searchResults) > 0 {
			inner += ui.SectionTitleStyle.Render("RESULTS") + "\n"
			for _, user := range m.searchResults {
				inner += fmt.Sprintf("  %s @%s\n",
					ui.InfoValueStyle.Render(user.FullName),
					ui.MutedStyle.Render(user.Username),
				)
			}
		}

		if m.err != nil {
			inner += "\n\n" + ui.ErrorTextStyle.Render("! "+m.err.Error())
		}
		content = inner
		footer = ui.FooterStyle.Render("enter: search/chat • esc: back")

	case stateChat:
		statusStr := ui.StatusLabelStyle.Foreground(ui.Success).Render("● online")
		if m.conn == nil {
			statusStr = ui.StatusLabelStyle.Foreground(ui.ErrorCol).Render("○ offline")
		}
		subHeader = ui.SubHeaderStyle.Render("chat / "+m.chatTarget+"  "+statusStr) + "\n"

		var chatContent string
		if len(m.chatMessages) == 0 {
			chatContent = ui.MutedStyle.Render("No messages yet. Say hello!")
		} else {
			// Show last 12 messages for better vertical space
			start := 0
			if len(m.chatMessages) > 12 {
				start = len(m.chatMessages) - 12
			}
			for i := start; i < len(m.chatMessages); i++ {
				msg := m.chatMessages[i]
				prefix := lipgloss.NewStyle().Foreground(ui.Secondary).Render("@" + msg.From + ":")
				if msg.IsMe {
					prefix = ui.SelectedStyle.Render("You:")
				}
				chatContent += fmt.Sprintf("%s %s\n", prefix, msg.Content)
			}
		}

		content = chatContent + "\n" + m.chatInput.View()
		footer = ui.FooterStyle.Render("enter: send • esc: back")

	case stateSettings:
		subHeader = ui.SubHeaderStyle.Render("settings") + "\n"

		inner := ui.SectionTitleStyle.Render("WORKSPACE") + "\n" + m.textInput.View() + "\n\n"
		inner += ui.SectionTitleStyle.Render("FULL NAME") + "\n" + m.nameInput.View() + "\n\n"

		if m.err != nil {
			inner += "\n\n" + ui.ErrorTextStyle.Render("! "+m.err.Error())
		}
		content = inner
		footer = ui.FooterStyle.Render("enter: save • x: self-destruct • esc: back")

	case stateConfirmPurge:
		subHeader = ui.SubHeaderStyle.Render("danger / self-destruct") + "\n"
		inner := ui.ErrorTextStyle.Render("ARE YOU ABSOLUTELY SURE?") + "\n\n"
		inner += "This will permanently:\n"
		inner += " - Delete your identity from the server\n"
		inner += " - Delete your local chat history\n"
		inner += " - Delete your cryptographic keys\n"
		inner += " - Reset this application\n\n"
		inner += ui.MutedStyle.Render("Press 'y' to confirm or 'n' to cancel")

		content = inner
		footer = ui.FooterStyle.Render("y/n")
	}

	body = ui.MainContainerStyle.Render(subHeader + content)

	return fmt.Sprintf("%s%s\n%s", header, body, footer)
}
