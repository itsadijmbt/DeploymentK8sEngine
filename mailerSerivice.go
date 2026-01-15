package main

import (
	"log"
	"os"
	"strings"

	"github.com/slack-go/slack"
)

type SlackMessage struct {
	Message     string
	Details     string
	MessageType string
}

const (
	MsgDeploymentSuccess  = "deployment-success"
	MsgDeploymentFailure  = "deployment-failure"
	MsgInternalSysFailure = "internal-system-failure"
	MsgNameSpaceError = "namepsace-issue";
)

func SlackNotifier(message SlackMessage) {

	// webhookURL := os.Getenv("SLACK_WEBHOOK_URL")

	go func() {

		// url -> data attachement -> create attachment -> create a msg -> send
		//get url
		webhook := getUrl()

		if webhook == "" {
			return
		}

		//get color , emoji
		color, emoji := getColor(message.MessageType)

		// crete an attachment via parsing of msg struct
		attachment := buildMessage(message, color, emoji)

		//create a message
		webhookMsg := &slack.WebhookMessage{
			Attachments: []slack.Attachment{attachment},
		}
		//send msg
		err := slack.PostWebhook(webhook, webhookMsg)
		if err != nil {
			log.Printf("Failed to send Slack notification: %v", err)
		}

	}()

}

func getUrl() string {

	url := os.Getenv("WEBHOOK_FOR_SLACK")

	if url == "" {
		log.Println("Webhook URL is not set")
		return ""
	}
	return url

}

func getColor(errorMsg string) (color, emoji string) {

	switch errorMsg {

	case MsgDeploymentSuccess:
		return "good", "✅"
	case MsgDeploymentFailure:
		return "danger", "❌"
	case MsgInternalSysFailure:
		return "warning", "⚠️"
	default:
		return "warning", "ℹ️"

	}

}

func buildMessage(msg SlackMessage, color, emoji string) slack.Attachment {

	feilds := parseDetails(msg.Details)

	return slack.Attachment{

		Color:  color,
		Title:  emoji + " " + msg.Message,
		Fields: feilds,
	}
}

func parseDetails(details string) []slack.AttachmentField {
	if details == "" {
		return []slack.AttachmentField{}
	}

	fields := []slack.AttachmentField{}
	lines := strings.Split(details, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		title := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		title = strings.Title(strings.ToLower(title))

		short := true
		titleLower := strings.ToLower(title)

		if titleLower == "error" || titleLower == "message" || len(value) > 50 {
			short = false
		}

		fields = append(fields, slack.AttachmentField{
			Title: title,
			Value: value,
			Short: short,
		})
	}
	return fields
}
