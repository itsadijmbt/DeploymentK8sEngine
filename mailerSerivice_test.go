package main

import (
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
)


func TestMain(m *testing.M) {

	godotenv.Load()


	code := m.Run()

	
	os.Exit(code)
}

// Test deployment success notification
func TestDeploymentSuccess(t *testing.T) {
	msg := SlackMessage{
		Message:     "Deployment Successful",
		Details:     "service:test-backend\nversion:v1.0.0\nnamespace:test",
		MessageType: MsgDeploymentSuccess,
	}

	// Send notification
	SlackNotifier(msg)

	// Wait for goroutine
	time.Sleep(2 * time.Second)

	t.Log("✅ Success notification sent - check Slack")
}

// Test deployment failure notification
func TestDeploymentFailure(t *testing.T) {
	msg := SlackMessage{
		Message:     "Deployment Failed",
		Details:     "service:test-frontend\nversion:v2.0.0\nnamespace:test\nerror:Connection timeout",
		MessageType: MsgDeploymentFailure,
	}

	SlackNotifier(msg)
	time.Sleep(2 * time.Second)

	t.Log("❌ Failure notification sent - check Slack")
}

// Test webhook URL loading
func TestGetUrl(t *testing.T) {
	url := getUrl()

	if url == "" {
		t.Fatal("WEBHOOK_FOR_SLACK not set in environment")
	}

	if len(url) < 50 {
		t.Fatal("Webhook URL seems too short")
	}

	t.Logf("✅ Webhook URL loaded: %s...", url[:50])
}

// Test parseDetails function
func TestParseDetails(t *testing.T) {
	details := "service:backend\nversion:v1.2.3\nerror:Something went wrong"

	fields := parseDetails(details)

	if len(fields) != 3 {
		t.Fatalf("Expected 3 fields, got %d", len(fields))
	}

	if fields[0].Title != "Service" || fields[0].Value != "backend" {
		t.Errorf("First field incorrect: %+v", fields[0])
	}

	if fields[2].Short != false {
		t.Error("Error field should be full width (Short=false)")
	}

	t.Log("✅ Parse details working correctly")
}

// Test color and emoji mapping
func TestGetColorAndEmoji(t *testing.T) {
	tests := []struct {
		msgType       string
		expectedColor string
		expectedEmoji string
	}{
		{MsgDeploymentSuccess, "good", "✅"},
		{MsgDeploymentFailure, "danger", "❌"},
		{"unknown-type", "warning", "ℹ️"},
	}

	for _, tt := range tests {
		color, emoji := getColor(tt.msgType)

		if color != tt.expectedColor {
			t.Errorf("Type %s: expected color %s, got %s",
				tt.msgType, tt.expectedColor, color)
		}

		if emoji != tt.expectedEmoji {
			t.Errorf("Type %s: expected emoji %s, got %s",
				tt.msgType, tt.expectedEmoji, emoji)
		}
	}

	t.Log("✅ Color and emoji mapping correct")
}
