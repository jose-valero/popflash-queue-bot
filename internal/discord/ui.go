package discord

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"
	"github.com/jose-valero/popflash-queue-bot/internal/queue"
)

func trimLabel(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	rs := []rune(s)
	if max <= 1 {
		return "…"
	}
	return string(rs[:max-1]) + "…"
}

func renderQueueEmbed(q *queue.Queue) *discordgo.MessageEmbed {
	if q == nil {
		return &discordgo.MessageEmbed{
			Title:       "❌ No hay cola activa",
			Description: "Usa `/startqueue` para crear una.",
			Color:       0xED4245, // rojo
		}
	}

	// Construimos la lista en un bloque de código para “cajita” visual
	var b strings.Builder
	if len(q.Players) == 0 {
		b.WriteString("_(No hay nadie llevándola)_")
	} else {
		for idx, p := range q.Players {
			fmt.Fprintf(&b, "%d) %s\n", idx+1, p.Username)
		}
	}

	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s (%d/%d)", q.Name, len(q.Players), q.Capacity),
		Description: b.String(),
		Color:       0xB069FF, // “blurple” de Discord
		Footer: &discordgo.MessageEmbedFooter{
			Text: "XCG BOT • /Chau saldras de la lista y no podremos guardar tu lugar",
		},
	}
}

// Construye componentes: fila de controles + filas de "Kick" por jugador.
// Respeta el límite de 5 filas totales (Discord) y 5 botones por fila.
func componentsWithKick(q *queue.Queue) []discordgo.MessageComponent {
	comps := componentsForQueue(q) // tu fila: La Llevo / Chau / Close / Reset
	if q == nil || len(q.Players) == 0 {
		return comps
	}

	// cuántas filas extra me quedan (máx 5 en total)
	remainingRows := 5 - len(comps)
	if remainingRows <= 0 {
		return comps
	}

	row := discordgo.ActionsRow{}
	rowsUsed := 0
	addRow := func() {
		if len(row.Components) > 0 {
			comps = append(comps, row)
			row = discordgo.ActionsRow{}
			rowsUsed++
		}
	}

	for _, p := range q.Players {
		btn := discordgo.Button{
			Label:    fmt.Sprintf("Kick %s", trimLabel(p.Username, 18)), // 👈 sin emoji, con nombre
			Style:    discordgo.DangerButton,
			CustomID: "kick_" + p.ID, // importante: el ID real va en el CustomID
		}
		row.Components = append(row.Components, btn)

		if len(row.Components) == 5 {
			addRow()
			if rowsUsed >= remainingRows {
				break
			}
		}
	}
	// última fila parcial
	if rowsUsed < remainingRows && len(row.Components) > 0 {
		comps = append(comps, row)
	}

	return comps
}

// NUEVO: misma fila pero con Join deshabilitado si la cola está llena
func componentsRowDisabled(disabled bool) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{Label: "La Llevo", Style: discordgo.PrimaryButton, CustomID: "queue_join", Disabled: disabled},
				discordgo.Button{Label: "Chau", Style: discordgo.SecondaryButton, CustomID: "queue_leave"},
				discordgo.Button{Label: "Close", Style: discordgo.DangerButton, CustomID: "queue_close"},
				discordgo.Button{Label: "Reset", Style: discordgo.DangerButton, CustomID: "queue_reset"},
			},
		},
	}
}

func componentsForQueue(q *queue.Queue) []discordgo.MessageComponent {
	if q == nil {
		return nil
	}
	full := len(q.Players) >= q.Capacity
	return componentsRowDisabled(full)
}
