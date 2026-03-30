package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	epp "github.com/onasunnymorning/eppclient"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

func main() {
	// Attempt to load .env file, overriding existing OS env vars like USER
	_ = godotenv.Overload()

	appToken := os.Getenv("SLACK_APP_TOKEN")
	botToken := os.Getenv("SLACK_BOT_TOKEN")

	if appToken == "" || !strings.HasPrefix(appToken, "xapp-") {
		log.Fatalf("SLACK_APP_TOKEN must be set and start with 'xapp-' for Socket Mode.")
	}
	if botToken == "" || !strings.HasPrefix(botToken, "xoxb-") {
		log.Fatalf("SLACK_BOT_TOKEN must be set and start with 'xoxb-' to send messages.")
	}

	cfg := &Config{
		Addr:     os.Getenv("ADDR"),
		User:     os.Getenv("USER"),
		Password: os.Getenv("PASSWORD"),
		TLS:      os.Getenv("TLS") == "true",
		Cert:     os.Getenv("CERT"),
		Key:      os.Getenv("KEY"),
		CACert:   os.Getenv("CACERT"),
	}

	if cfg.Addr == "" || cfg.User == "" {
		log.Fatalf("EPP config missing ADDR or USER in .env")
	}

	api := slack.New(
		botToken,
		slack.OptionAppLevelToken(appToken),
	)

	client := socketmode.New(
		api,
		socketmode.OptionDebug(false),
		socketmode.OptionLog(log.New(os.Stdout, "socketmode: ", log.Lshortfile|log.LstdFlags)),
	)

	// Goroutine to handle events
	go func() {
		for evt := range client.Events {
			switch evt.Type {
			case socketmode.EventTypeSlashCommand:
				cmd, ok := evt.Data.(slack.SlashCommand)
				if !ok {
					log.Printf("Ignored %+v\n", evt)
					continue
				}

				// Acknowledge the slash command so Slack knows we received it
				client.Ack(*evt.Request)

				if cmd.Command == "/epp" {
					go handleEppCommand(api, cmd, cfg)
				} else {
					log.Printf("Unknown command received: %s", cmd.Command)
				}

			default:
				// ignore other events
			}
		}
	}()

	fmt.Println("EPP Slack Bot started via Socket Mode! Waiting for commands...")
	err := client.Run()
	if err != nil {
		log.Fatalf("Socket mode run error: %v", err)
	}
}

func handleEppCommand(api *slack.Client, cmd slack.SlashCommand, cfg *Config) {
	args := strings.Fields(cmd.Text)
	if len(args) < 2 || args[0] != "check" {
		replyError(api, cmd.ChannelID, cmd.UserID, "Usage: `/epp check <domain>`")
		return
	}

	domains := args[1:]
	// Send "Checking..." provisional message if desired, or just wait.
	// We'll post an ephemeral message as immediate feedback while EPP runs.
	_, _, _ = api.PostMessage(cmd.ChannelID, 
		slack.MsgOptionText(fmt.Sprintf("Checking availability for %s...", strings.Join(domains, ", ")), false),
		slack.MsgOptionPostEphemeral(cmd.UserID),
	)

	// Call EPP
	dcr, err := checkDomain(cfg, domains)
	if err != nil {
		replyError(api, cmd.ChannelID, cmd.UserID, fmt.Sprintf("Error running check: %v", err))
		return
	}

	// Format results
	blocks := buildResultBlocks(dcr, cfg)
	
	// Assuming you want the result visible to everyone in the channel, we post back normally.
	_, _, err = api.PostMessage(cmd.ChannelID, slack.MsgOptionBlocks(blocks...))
	if err != nil {
		log.Printf("failed posting message: %v", err)
	}
}

func replyError(api *slack.Client, channelID, userID, text string) {
	_, _, _ = api.PostMessage(
		channelID,
		slack.MsgOptionText("⚠️ "+text, false),
		slack.MsgOptionPostEphemeral(userID),
	)
}

func buildResultBlocks(dcr *epp.DomainCheckResponse, cfg *Config) []slack.Block {
	if dcr == nil || len(dcr.Checks) == 0 {
		return []slack.Block{
			slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "No domains checked.", false, false), nil, nil),
		}
	}

	var blocks []slack.Block
	blocks = append(blocks, slack.NewHeaderBlock(slack.NewTextBlockObject("plain_text", "Domain Check Results 📡", false, false)))

	// Context block for server and user used
	blocks = append(blocks, slack.NewContextBlock("", slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("Server: *%s* | User: *%s*", cfg.Addr, cfg.User), false, false)))

	for _, c := range dcr.Checks {
		var icon, statusText string
		if c.Available {
			icon = "✅"
			statusText = "*Available*"
		} else {
			icon = "❌"
			statusText = "*Unavailable*"
		}

		line := fmt.Sprintf("%s  `%s` %s", icon, c.Domain, statusText)
		if c.Reason != "" {
			line += fmt.Sprintf("\n> _Reason: %s_", c.Reason)
		}
		
		blocks = append(blocks, slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", line, false, false), nil, nil))
	}

	blocks = append(blocks, slack.NewDividerBlock())

	// Incorporate fee info similarly to the CLI if present
	if len(dcr.Charges) > 0 {
		feeText := "*Fees & Pricing:*\n"
		for _, cg := range dcr.Charges {
			if len(cg.Fees) == 0 && cg.Category == "" {
				continue
			}

			// Add basic domain price header
			feeText += fmt.Sprintf("• `%s`", cg.Domain)
			if cg.Category != "" {
				feeText += fmt.Sprintf(" (Category: %s)", cg.Category)
			}
			feeText += "\n"

			for _, f := range cg.Fees {
				curr := cg.Currency
				
				if curr != "" {
					feeText += fmt.Sprintf("    _%s_: %s %s", f.Name, f.Amount, curr)
				} else {
					feeText += fmt.Sprintf("    _%s_: %s", f.Name, f.Amount)
				}
				
				var attrs []string
				if f.Standard {
					attrs = append(attrs, "standard")
				}
				if f.Refundable {
					attrs = append(attrs, "refundable")
				}
				if len(attrs) > 0 {
					feeText += fmt.Sprintf(" _(%s)_", strings.Join(attrs, ", "))
				}
				feeText += "\n"
			}
		}
		
		blocks = append(blocks, slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", feeText, false, false), nil, nil))
	}

	return blocks
}
