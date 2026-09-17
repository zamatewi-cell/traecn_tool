package traeapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_MockAPIs(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case PathGetUserInfo:
			json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 0,
				"data": map[string]interface{}{
					"UserID":             "123456",
					"ScreenName":         "TraeDeveloper",
					"NonPlainTextEmail": "dev@example.com",
					"AvatarUrl":          "https://example.com/avatar.png",
				},
			})
		case PathPayStatusV2:
			json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 0,
				"data": map[string]interface{}{
					"user_pay_identity_str": "Pro",
					"is_pay_freshman":      false,
					"detail": map[string]interface{}{
						"subscription_renew_time": 1750000000,
					},
				},
			})
		case PathEntUsageV2:
			json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 0,
				"data": map[string]interface{}{
					"user_entitlement_pack_list": []map[string]interface{}{
						{
							"entitlement_pack_id":   "pack_1",
							"entitlement_pack_name": "Pro 月度权益包",
							"entitlement_base_info": map[string]interface{}{
								"product_type": 1,
								"quota": map[string]interface{}{
									"total_quota": 500,
									"unit":        "次",
								},
							},
							"usage": map[string]interface{}{
								"credits_amount": 42,
							},
						},
					},
				},
			})
		case PathCheckinStatus:
			json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 0,
				"data": map[string]interface{}{
					"checked_in": false,
					"enable":     true,
					"credits":    1280,
				},
			})
		case PathCheckinClaim:
			json.NewEncoder(w).Encode(map[string]interface{}{
				"code":    0,
				"message": "success",
				"data": map[string]interface{}{
					"credits_earned": 20,
					"credits":        1300,
				},
			})
		case PathUsageGroupBySession:
			json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 0,
				"data": map[string]interface{}{
					"total": 1,
					"user_usage_group_by_sessions": []map[string]interface{}{
						{
							"session_id":         "sess_abc",
							"session_start_time": 1720000000,
							"model_name":         "DeepSeek-V4.1-Flash",
							"credits_float":      1.5,
							"prompt_tokens":      100,
							"completion_tokens":  200,
							"user_input_preview": "Hello Trae",
						},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	ctx := context.Background()

	// 1. UserInfo
	user, err := client.GetUserInfo(ctx, "mock_token")
	if err != nil {
		t.Fatalf("GetUserInfo failed: %v", err)
	}
	if user.Nickname != "TraeDeveloper" || user.UserID != "123456" {
		t.Errorf("Unexpected user: %+v", user)
	}

	// 2. PayStatus
	pay, err := client.GetPayStatus(ctx, "mock_token")
	if err != nil {
		t.Fatalf("GetPayStatus failed: %v", err)
	}
	if pay.PlanType != "Pro" || pay.ExpireAt != 1750000000 {
		t.Errorf("Unexpected pay: %+v", pay)
	}

	// 3. Entitlements
	packs, err := client.GetEntitlements(ctx, "mock_token")
	if err != nil {
		t.Fatalf("GetEntitlements failed: %v", err)
	}
	if len(packs) != 1 || packs[0].TotalQuota != 500 || packs[0].UsedQuota != 42 {
		t.Errorf("Unexpected packs: %+v", packs)
	}

	// 4. CheckinStatus
	checkin, err := client.GetCheckinStatus(ctx, "mock_token", "did_test")
	if err != nil {
		t.Fatalf("GetCheckinStatus failed: %v", err)
	}
	if checkin.CheckedIn || !checkin.CanCheckin || checkin.Credits != 1280 {
		t.Errorf("Unexpected checkin: %+v", checkin)
	}

	// 5. ClaimCheckin
	claim, err := client.ClaimCheckin(ctx, "mock_token", "did_test")
	if err != nil {
		t.Fatalf("ClaimCheckin failed: %v", err)
	}
	if !claim.Success || claim.CreditsEarned != 20 || claim.NewBalance != 1300 {
		t.Errorf("Unexpected claim: %+v", claim)
	}

	// 6. UsageRecords
	records, total, err := client.GetUsageRecords(ctx, "mock_token", 1, 10)
	if err != nil {
		t.Fatalf("GetUsageRecords failed: %v", err)
	}
	if total != 1 || len(records) != 1 || records[0].ModelName != "DeepSeek-V4.1-Flash" {
		t.Errorf("Unexpected records: %+v", records)
	}

	// 7. FullProfile
	profile, err := client.GetFullProfile(ctx, "mock_token", "did_test")
	if err != nil {
		t.Fatalf("GetFullProfile failed: %v", err)
	}
	if profile.UserInfo.Nickname != "TraeDeveloper" || profile.Checkin.Credits != 1280 {
		t.Errorf("Unexpected full profile: %+v", profile)
	}
}
