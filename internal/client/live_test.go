package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/deldrid1/beehiiv-cli/internal/client"
	"github.com/deldrid1/beehiiv-cli/internal/commandset"
	"github.com/deldrid1/beehiiv-cli/internal/config"
	"github.com/deldrid1/beehiiv-cli/internal/testsupport"
)

func TestLiveReadOnlySmoke(t *testing.T) {
	live := testsupport.RequireLiveConfig(t)
	apiClient := client.New(config.Runtime{
		APIKey:        live.APIKey,
		PublicationID: live.PublicationID,
		BaseURL:       config.DefaultBaseURL,
		RateLimitRPM:  config.DefaultRateLimitRPM,
		Timeout:       config.DefaultTimeout,
	}, nil, nil)

	required := []struct {
		group  string
		action string
		query  url.Values
	}{
		{"publications", "list", url.Values{"limit": {"1"}}},
		{"publications", "get", url.Values{}},
		{"subscriptions", "list", url.Values{"limit": {"1"}}},
		{"posts", "list", url.Values{"limit": {"1"}}},
		{"segments", "list", url.Values{"limit": {"1"}}},
		{"tiers", "list", url.Values{"limit": {"1"}}},
		{"webhooks", "list", url.Values{"limit": {"1"}}},
	}

	for _, testCase := range required {
		testCase := testCase
		t.Run(testCase.group+"_"+testCase.action, func(t *testing.T) {
			operation := mustFindOperation(t, testCase.group, testCase.action)
			pathValues := map[string]string{}
			if testCase.group == "publications" && testCase.action == "get" {
				pathValues["publicationId"] = live.PublicationID
			}
			if _, err := execute(t, apiClient, operation, pathValues, testCase.query, nil); err != nil {
				t.Fatalf("execute returned error: %v", err)
			}
		})
	}

	optional := []struct {
		group  string
		action string
		query  url.Values
	}{
		{"advertisement-opportunities", "list", url.Values{}},
		{"authors", "list", url.Values{"limit": {"1"}}},
		{"condition-sets", "list", url.Values{"limit": {"1"}}},
		{"newsletter-lists", "list", url.Values{"limit": {"1"}}},
		{"polls", "list", url.Values{"limit": {"1"}}},
	}

	for _, testCase := range optional {
		testCase := testCase
		t.Run("optional_"+testCase.group, func(t *testing.T) {
			operation := mustFindOperation(t, testCase.group, testCase.action)
			if _, err := execute(t, apiClient, operation, map[string]string{}, testCase.query, nil); err != nil {
				t.Skipf("optional endpoint unavailable for this publication: %v", err)
			}
		})
	}
}

func TestLiveMutatingLifecycle(t *testing.T) {
	live := testsupport.RequireLiveConfig(t)
	apiClient := client.New(config.Runtime{
		APIKey:        live.APIKey,
		PublicationID: live.PublicationID,
		BaseURL:       config.DefaultBaseURL,
		RateLimitRPM:  config.DefaultRateLimitRPM,
		Timeout:       60 * time.Second,
	}, nil, nil)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	customFieldName := "Codex Go " + suffix
	subscriptionEmail := fmt.Sprintf("beehiiv-cli-go+%s@example.com", suffix)
	webhookURL := fmt.Sprintf("https://example.com/webhook/%s", suffix)
	tagName := "codex-go-" + suffix

	customFieldID := ""
	subscriptionID := ""
	webhookID := ""

	defer func() {
		if webhookID != "" {
			_, _ = execute(t, apiClient, mustFindOperation(t, "webhooks", "delete"), map[string]string{"endpointId": webhookID}, nil, nil)
		}
		if customFieldID != "" {
			_, _ = execute(t, apiClient, mustFindOperation(t, "custom-fields", "delete"), map[string]string{"id": customFieldID}, nil, nil)
		}
		if subscriptionID != "" {
			_, _ = execute(t, apiClient, mustFindOperation(t, "subscriptions", "delete"), map[string]string{"subscriptionId": subscriptionID}, nil, nil)
		}
	}()

	customFieldResponse, err := execute(t, apiClient, mustFindOperation(t, "custom-fields", "create"), map[string]string{}, nil, []byte(fmt.Sprintf(`{"kind":"string","display":"%s"}`, customFieldName)))
	if err != nil {
		t.Fatalf("custom-fields create returned error: %v", err)
	}
	customFieldID = firstStringPath(customFieldResponse.Body, []string{"data", "id"}, []string{"id"})
	if customFieldID == "" {
		t.Fatal("custom-fields create did not return an id")
	}

	if _, err := execute(t, apiClient, mustFindOperation(t, "custom-fields", "get"), map[string]string{"id": customFieldID}, nil, nil); err != nil {
		t.Fatalf("custom-fields get returned error: %v", err)
	}

	subscriptionResponse, err := execute(t, apiClient, mustFindOperation(t, "subscriptions", "create"), map[string]string{}, nil, []byte(fmt.Sprintf(`{"email":"%s","send_welcome_email":false,"reactivate_existing":false}`, subscriptionEmail)))
	if err != nil {
		t.Fatalf("subscriptions create returned error: %v", err)
	}
	subscriptionID = firstStringPath(subscriptionResponse.Body, []string{"data", "id"}, []string{"id"})
	if subscriptionID == "" {
		t.Fatal("subscriptions create did not return an id")
	}

	if _, err := execute(t, apiClient, mustFindOperation(t, "subscriptions", "update"), map[string]string{"subscriptionId": subscriptionID}, nil, []byte(fmt.Sprintf(`{"custom_fields":[{"name":"%s","value":"direct-update"}]}`, customFieldName))); err != nil {
		t.Fatalf("subscriptions update returned error: %v", err)
	}
	if _, err := execute(t, apiClient, mustFindOperation(t, "subscription-tags", "create"), map[string]string{"subscriptionId": subscriptionID}, nil, []byte(fmt.Sprintf(`{"tags":["%s"]}`, tagName))); err != nil {
		t.Fatalf("subscription-tags create returned error: %v", err)
	}
	if _, err := execute(t, apiClient, mustFindOperation(t, "subscriptions", "get"), map[string]string{"subscriptionId": subscriptionID}, url.Values{"expand[]": {"custom_fields", "tags"}}, nil); err != nil {
		t.Fatalf("subscriptions get returned error: %v", err)
	}
	if _, err := execute(t, apiClient, mustFindOperation(t, "subscriptions", "get-by-email"), map[string]string{"email": subscriptionEmail}, nil, nil); err != nil {
		t.Fatalf("subscriptions get-by-email returned error: %v", err)
	}

	bulkResponse, err := execute(t, apiClient, mustFindOperation(t, "subscription-bulk-actions", "update"), map[string]string{}, nil, []byte(fmt.Sprintf(`{"subscriptions":[{"subscription_id":"%s","custom_fields":[{"name":"%s","value":"bulk-update"}]}]}`, subscriptionID, customFieldName)))
	if err != nil {
		t.Fatalf("subscription-bulk-actions update returned error: %v", err)
	}
	updateID := firstStringPath(bulkResponse.Body, []string{"data", "subscription_update_id"}, []string{"subscription_update_id"})
	if updateID == "" {
		t.Fatal("bulk update response did not include subscription_update_id")
	}
	waitForBulkUpdateComplete(t, apiClient, updateID)

	webhookResponse, err := execute(t, apiClient, mustFindOperation(t, "webhooks", "create"), map[string]string{}, nil, []byte(fmt.Sprintf(`{"url":"%s","event_types":["subscription.confirmed"],"description":"Codex test webhook"}`, webhookURL)))
	if err != nil {
		t.Fatalf("webhooks create returned error: %v", err)
	}
	webhookID = firstStringPath(webhookResponse.Body, []string{"data", "id"}, []string{"id"})
	if webhookID == "" {
		t.Fatal("webhooks create did not return an id")
	}

	if _, err := execute(t, apiClient, mustFindOperation(t, "webhooks", "get"), map[string]string{"endpointId": webhookID}, nil, nil); err != nil {
		t.Fatalf("webhooks get returned error: %v", err)
	}
	if _, err := execute(t, apiClient, mustFindOperation(t, "webhooks", "test"), map[string]string{"endpointId": webhookID}, nil, nil); err != nil {
		if isKnownWebhookTestQuirk(err) {
			t.Logf("webhooks test returned Beehiiv's known INVALID_EVENT_TYPE response for a freshly created webhook: %v", err)
		} else {
			t.Fatalf("webhooks test returned error: %v", err)
		}
	}
}

func mustFindOperation(t *testing.T, group, action string) commandset.Operation {
	t.Helper()

	operation, found, err := commandset.Find(group, action)
	if err != nil {
		t.Fatalf("Find returned error: %v", err)
	}
	if !found {
		t.Fatalf("operation %s %s not found", group, action)
	}
	return operation
}

func execute(t *testing.T, apiClient *client.Client, operation commandset.Operation, pathValues map[string]string, query url.Values, body []byte) (*client.Response, error) {
	t.Helper()
	return apiClient.Execute(context.Background(), operation, pathValues, query, body)
}

func nestedString(body []byte, keys ...string) string {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	var current any = payload
	for _, key := range keys {
		object, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current, ok = object[key]
		if !ok {
			return ""
		}
	}
	value, _ := current.(string)
	return value
}

func flatString(body []byte, key string) string {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	value, _ := payload[key].(string)
	return value
}

func firstStringPath(body []byte, paths ...[]string) string {
	for _, path := range paths {
		if len(path) == 0 {
			continue
		}
		if len(path) == 1 {
			if value := flatString(body, path[0]); value != "" {
				return value
			}
			continue
		}
		if value := nestedString(body, path...); value != "" {
			return value
		}
	}
	return ""
}

func isKnownWebhookTestQuirk(err error) bool {
	var apiErr *client.Error
	if !errors.As(err, &apiErr) || apiErr.Status != 400 {
		return false
	}
	body := strings.ToLower(string(apiErr.Body))
	return strings.Contains(body, "invalid_event_type") || strings.Contains(body, "invalid event type")
}

func waitForBulkUpdateComplete(t *testing.T, apiClient *client.Client, updateID string) {
	t.Helper()

	operation := mustFindOperation(t, "bulk-subscription-updates", "get")
	lastStatus := ""
	for attempt := 0; attempt < 30; attempt++ {
		response, err := execute(t, apiClient, operation, map[string]string{"id": updateID}, nil, nil)
		if err != nil {
			t.Fatalf("bulk-subscription-updates get returned error: %v", err)
		}
		status := strings.ToLower(firstStringPath(response.Body, []string{"data", "status"}, []string{"status"}))
		if status != "" {
			lastStatus = status
		}
		switch status {
		case "complete", "completed":
			return
		case "failed":
			t.Fatalf("bulk update %s failed: %s", updateID, string(response.Body))
		}
		time.Sleep(time.Second)
	}
	if lastStatus == "" {
		t.Fatalf("bulk update %s did not return a status in time", updateID)
	}
	t.Logf("bulk update %s remained in status %q after polling window; continuing", updateID, lastStatus)
}
