package discord

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

func msgWithText(s string) *discordgo.Message {
	return &discordgo.Message{Content: s}
}

func msgWithEmbed(e *discordgo.MessageEmbed) *discordgo.Message {
	return &discordgo.Message{Embeds: []*discordgo.MessageEmbed{e}}
}

func TestDetectPFTriggers_Text(t *testing.T) {
	st, fin := detectPFTriggers(msgWithText("PopFlash Match 123 started"))
	if !st || fin {
		t.Fatalf("want started=true finished=false; got %v %v", st, fin)
	}

	st, fin = detectPFTriggers(msgWithText("..MATCH FINISHED.."))
	if st || !fin {
		t.Fatalf("want started=false finished=true; got %v %v", st, fin)
	}
}

func TestDetectPFTriggers_EmbedTitleAndDesc(t *testing.T) {
	e := &discordgo.MessageEmbed{
		Title:       "PopFlash Match 999 Started",
		Description: "whatever",
	}
	st, fin := detectPFTriggers(msgWithEmbed(e))
	if !st || fin {
		t.Fatalf("title 'started' must trigger; got %v %v", st, fin)
	}

	e2 := &discordgo.MessageEmbed{
		Title:       "PopFlash Match 888",
		Description: "This match FINISHED just now",
	}
	st, fin = detectPFTriggers(msgWithEmbed(e2))
	if st || !fin {
		t.Fatalf("description 'finished' must trigger; got %v %v", st, fin)
	}
}

func TestDetectPFTriggers_EmbedFieldsAuthorFooter(t *testing.T) {
	e := &discordgo.MessageEmbed{
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Console", Value: "connect ... ; password popflash"},
			{Name: "Status", Value: "Match Started"},
		},
		Author: &discordgo.MessageEmbedAuthor{Name: "PopFlash Bot"},
		Footer: &discordgo.MessageEmbedFooter{Text: "some footer"},
	}
	st, fin := detectPFTriggers(msgWithEmbed(e))
	if !st || fin {
		t.Fatalf("field 'Started' must trigger; got %v %v", st, fin)
	}

	e2 := &discordgo.MessageEmbed{
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Status", Value: "Match Finished"},
		},
	}
	st, fin = detectPFTriggers(msgWithEmbed(e2))
	if st || !fin {
		t.Fatalf("field 'Finished' must trigger; got %v %v", st, fin)
	}
}

func TestDetectPFTriggers_NoiseDoesNotTrigger(t *testing.T) {
	e := &discordgo.MessageEmbed{
		Title:       "PopFlash something",
		Description: "random text",
	}
	st, fin := detectPFTriggers(msgWithEmbed(e))
	if st || fin {
		t.Fatalf("should not trigger; got %v %v", st, fin)
	}
}

func TestDetectPFTriggers_BothWords(t *testing.T) {
	e := &discordgo.MessageEmbed{
		Description: "match started ... later match finished",
	}
	st, fin := detectPFTriggers(msgWithEmbed(e))
	if !st || !fin {
		t.Fatalf("both should be true; got %v %v", st, fin)
	}
}
