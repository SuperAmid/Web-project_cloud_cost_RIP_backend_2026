package handler

import (
	"net/http/httptest"
	"strings"
	"testing"

	"provider_routers/internal/serializer"
)

func TestDecodeJSONRejectsSystemFields(t *testing.T) {
	for _, field := range []string{"id", "status", "created_by_id", "created_at", "published_at"} {
		body := `{"description":"x","bandwidth_mbps":1,"cost_rub":1,"` + field + `":1}`
		request := httptest.NewRequest("PUT", "/api/provider_routers/1/publish", strings.NewReader(body))
		var dst serializer.ProviderRouterPublishRequest
		if err := decodeJSON(request, &dst); err == nil {
			t.Fatalf("поле %s должно отклоняться", field)
		}
	}
}

func TestValidatePublish(t *testing.T) {
	text, bandwidth, cost, badType := "Маршрутизатор", 1000, 50000, "core"
	if _, err := validatePublish(serializer.ProviderRouterPublishRequest{Description: &text, BandwidthMbps: &bandwidth, CostRub: &cost}); err != nil {
		t.Fatalf("корректная публикация отклонена: %v", err)
	}
	if _, err := validatePublish(serializer.ProviderRouterPublishRequest{Description: &text, BandwidthMbps: &bandwidth}); err == nil {
		t.Fatal("без cost_rub публикация должна отклоняться")
	}
	if _, err := validatePublish(serializer.ProviderRouterPublishRequest{Description: &text, BandwidthMbps: &bandwidth, CostRub: &cost, RouterType: &badType}); err == nil {
		t.Fatal("неизвестный router_type должен отклоняться")
	}
}
