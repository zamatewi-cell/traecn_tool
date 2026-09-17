package traeapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	DefaultOriginCN = "https://api.trae.cn"
	DefaultTimeout  = 15 * time.Second

	PathGetUserInfo         = "/cloudide/api/v3/trae/GetUserInfo"
	PathPayStatusV2         = "/trae/api/v2/pay/ide_user_pay_status"
	PathPayStatusV1         = "/trae/api/v1/pay/ide_user_pay_status"
	PathEntUsageV2          = "/trae/api/v2/pay/ide_user_ent_usage"
	PathEntUsageV1          = "/trae/api/v1/pay/ide_user_ent_usage"
	PathCheckinStatus       = "/trae/api/v2/ug/checkin_credits/status"
	PathCheckinClaim        = "/trae/api/v2/ug/checkin_credits/claim"
	PathUsageGroupBySession = "/trae/api/v1/pay/query_user_usage_group_by_session"
)

// UserInfo represents Trae account basic profile
type UserInfo struct {
	UserID    string `json:"user_id"`
	Nickname  string `json:"nickname"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// PayStatus represents user subscription/billing plan
type PayStatus struct {
	PlanType      string `json:"plan_type"`
	ExpireAt      int64  `json:"expire_at,omitempty"`
	IsPayFreshman bool   `json:"is_pay_freshman"`
}

// UsageSummary represents account total credits pool
type UsageSummary struct {
	TotalAmount      float64 `json:"total_amount"`
	ConsumedAmount   float64 `json:"consumed_amount"`
	RemainingAmount  float64 `json:"remaining_amount"`
	ConsumptionRatio float64 `json:"consumption_ratio"`
}

// EntitlementPack represents a quota package (e.g. 500 requests/month)
type EntitlementPack struct {
	PackID     string  `json:"pack_id"`
	PackName   string  `json:"pack_name"`
	PackDesc   string  `json:"pack_desc"`
	Category   string  `json:"category"` // "general" (通用积分) 或 "work" (Work专属积分)
	TotalQuota float64 `json:"total_quota"`
	UsedQuota  float64 `json:"used_quota"`
	Unit       string  `json:"unit"`
	ExpireTime int64   `json:"expire_time"`
	Status     int     `json:"status"`
}

// CheckinInfo represents daily checkin status & credits balance
type CheckinInfo struct {
	CheckedIn    bool    `json:"checked_in"`
	CanCheckin   bool    `json:"can_checkin"`
	Credits      float64 `json:"credits"`
	ExtraCredits float64 `json:"extra_credits"`
}

// CheckinResult is the return value of ClaimCheckin
type CheckinResult struct {
	Success        bool    `json:"success"`
	AlreadyClaimed bool    `json:"already_claimed"`
	CreditsEarned  float64 `json:"credits_earned"`
	NewBalance     float64 `json:"new_balance"`
	Message        string  `json:"message"`
}

// UsageSessionRecord represents a session usage record from Trae cloud
type UsageSessionRecord struct {
	SessionID        string  `json:"session_id"`
	StartTime        int64   `json:"start_time"`
	EndTime          int64   `json:"end_time"`
	ModelName        string  `json:"model_name"`
	ProductName      string  `json:"product_name"`
	CreditsConsumed  float64 `json:"credits_consumed"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	TotalTokens      int64   `json:"total_tokens"`
	Preview          string  `json:"preview"`
}

// ProfileResponse bundles all profile details together
type ProfileResponse struct {
	UserInfo         UserInfo          `json:"user_info"`
	PayStatus        PayStatus         `json:"pay_status"`
	UsageSummary     UsageSummary      `json:"usage_summary"`
	SpendableCredits float64           `json:"spendable_credits"`
	GeneralCredits   float64           `json:"general_credits"`
	WorkCredits      float64           `json:"work_credits"`
	TotalCredits     float64           `json:"total_credits"`
	Entitlements     []EntitlementPack `json:"entitlements"`
	Checkin          CheckinInfo       `json:"checkin"`
}

// Client manages calls to Trae backend APIs
type Client struct {
	httpClient *http.Client
	origin     string
}

// NewClient creates a Trae OpenAPI client
func NewClient(origin string) *Client {
	if origin == "" {
		origin = DefaultOriginCN
	}
	return &Client{
		httpClient: &http.Client{Timeout: DefaultTimeout},
		origin:     strings.TrimRight(origin, "/"),
	}
}

func (c *Client) doRequest(ctx context.Context, method, path string, headers map[string]string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	reqURL := c.origin + path
	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Trae/1.0.0 trae-proxy")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request %s: %w", path, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response %s: %w", path, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status %d: %s", resp.StatusCode, string(respBytes))
	}

	return respBytes, nil
}

// GetUserInfo fetches current user nickname, avatar, email, etc.
func (c *Client) GetUserInfo(ctx context.Context, token string) (*UserInfo, error) {
	headers := map[string]string{
		"Authorization":    "Bearer " + token,
		"x-cloudide-token": token,
	}

	raw, err := c.doRequest(ctx, http.MethodPost, PathGetUserInfo, headers, map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	var root struct {
		Code   int             `json:"code"`
		Msg    string          `json:"message"`
		Data   json.RawMessage `json:"data"`
		Result json.RawMessage `json:"Result"`
	}
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, err
	}

	target := root.Data
	if len(target) == 0 {
		target = root.Result
	}
	if len(target) == 0 {
		target = raw
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(target, &parsed); err != nil {
		return nil, err
	}

	info := &UserInfo{}
	if v, ok := parsed["ScreenName"].(string); ok && v != "" {
		info.Nickname = v
	} else if v, ok := parsed["Nickname"].(string); ok && v != "" {
		info.Nickname = v
	} else if v, ok := parsed["nickname"].(string); ok && v != "" {
		info.Nickname = v
	}

	if v, ok := parsed["NonPlainTextEmail"].(string); ok && v != "" {
		info.Email = v
	} else if v, ok := parsed["Email"].(string); ok && v != "" {
		info.Email = v
	} else if v, ok := parsed["email"].(string); ok && v != "" {
		info.Email = v
	}

	if v, ok := parsed["AvatarUrl"].(string); ok && v != "" {
		info.AvatarURL = v
	} else if v, ok := parsed["avatar_url"].(string); ok && v != "" {
		info.AvatarURL = v
	}

	if v, ok := parsed["UserID"]; ok {
		info.UserID = fmt.Sprintf("%v", v)
	} else if v, ok := parsed["user_id"]; ok {
		info.UserID = fmt.Sprintf("%v", v)
	}

	return info, nil
}

// GetPayStatus retrieves plan status (Free/Pro) and renewal time
func (c *Client) GetPayStatus(ctx context.Context, token string) (*PayStatus, error) {
	headers := map[string]string{
		"Authorization": "Cloud-IDE-JWT " + token,
	}

	paths := []string{PathPayStatusV2, PathPayStatusV1}
	for _, p := range paths {
		raw, err := c.doRequest(ctx, http.MethodPost, p, headers, map[string]interface{}{})
		if err != nil {
			continue
		}

		var parsed struct {
			Code               int    `json:"code"`
			UserPayIdentityStr string `json:"user_pay_identity_str"`
			IsPayFreshman      bool   `json:"is_pay_freshman"`
			Detail             struct {
				SubscriptionRenewTime int64 `json:"subscription_renew_time"`
			} `json:"detail"`
			Data struct {
				UserPayIdentityStr string `json:"user_pay_identity_str"`
				IsPayFreshman      bool   `json:"is_pay_freshman"`
				Detail             struct {
					SubscriptionRenewTime int64 `json:"subscription_renew_time"`
				} `json:"detail"`
			} `json:"data"`
			Result struct {
				UserPayIdentityStr string `json:"user_pay_identity_str"`
				IsPayFreshman      bool   `json:"is_pay_freshman"`
				Detail             struct {
					SubscriptionRenewTime int64 `json:"subscription_renew_time"`
				} `json:"detail"`
			} `json:"result"`
		}
		if err := json.Unmarshal(raw, &parsed); err == nil {
			plan := parsed.UserPayIdentityStr
			renew := parsed.Detail.SubscriptionRenewTime
			freshman := parsed.IsPayFreshman

			if plan == "" {
				plan = parsed.Data.UserPayIdentityStr
				renew = parsed.Data.Detail.SubscriptionRenewTime
				freshman = parsed.Data.IsPayFreshman
			}
			if plan == "" {
				plan = parsed.Result.UserPayIdentityStr
				renew = parsed.Result.Detail.SubscriptionRenewTime
				freshman = parsed.Result.IsPayFreshman
			}
			if plan != "" {
				return &PayStatus{
					PlanType:      plan,
					ExpireAt:      renew,
					IsPayFreshman: freshman,
				}, nil
			}
		}
	}

	return &PayStatus{PlanType: "Free"}, nil
}

// GetEntitlementsResult holds both summary and pack details
type GetEntitlementsResult struct {
	Summary UsageSummary
	Packs   []EntitlementPack
}

// GetEntitlements retrieves quota package usage & total amount
func (c *Client) GetEntitlements(ctx context.Context, token string) (*GetEntitlementsResult, error) {
	headers := map[string]string{
		"Authorization": "Cloud-IDE-JWT " + token,
	}

	paths := []string{PathEntUsageV2, PathEntUsageV1}
	for _, p := range paths {
		raw, err := c.doRequest(ctx, http.MethodPost, p, headers, map[string]interface{}{"require_usage": true})
		if err != nil {
			continue
		}

		type packItem struct {
			PackID              string  `json:"entitlement_pack_id"`
			PackName            string  `json:"entitlement_pack_name"`
			GroupName           string  `json:"group_name"`
			DisplayDesc         string  `json:"display_desc"`
			PackDesc            string  `json:"entitlement_pack_desc"`
			Status              int     `json:"status"`
			ExpireTime          int64   `json:"expire_time"`
			TotalQuota          float64 `json:"total_quota"`
			Quota               float64 `json:"quota"`
			CreditsLimit        float64 `json:"credits_limit"`
			EntitlementBaseInfo struct {
				ProductType       int `json:"product_type"`
				ProductID         int `json:"product_id"`
				AvailableEndpoint int `json:"available_endpoint"`
				Quota             struct {
					TotalQuota   float64 `json:"total_quota"`
					CreditsLimit float64 `json:"credits_limit"`
					Unit         string  `json:"unit"`
				} `json:"quota"`
			} `json:"entitlement_base_info"`
			Usage struct {
				CreditsAmount float64 `json:"credits_amount"`
				UsedQuota     float64 `json:"used_quota"`
			} `json:"usage"`
		}

		var parsed struct {
			Code         int `json:"code"`
			UsageSummary struct {
				ConsumedAmount   float64 `json:"consumed_amount"`
				ConsumptionRatio float64 `json:"consumption_ratio"`
				TotalAmount      float64 `json:"total_amount"`
			} `json:"usage_summary"`
			UserEntitlementPackList []packItem `json:"user_entitlement_pack_list"`
			Data                    struct {
				UsageSummary struct {
					ConsumedAmount   float64 `json:"consumed_amount"`
					ConsumptionRatio float64 `json:"consumption_ratio"`
					TotalAmount      float64 `json:"total_amount"`
				} `json:"usage_summary"`
				UserEntitlementPackList []packItem `json:"user_entitlement_pack_list"`
			} `json:"data"`
		}

		if err := json.Unmarshal(raw, &parsed); err == nil {
			// Extract summary
			summary := UsageSummary{
				TotalAmount:      parsed.UsageSummary.TotalAmount,
				ConsumedAmount:   parsed.UsageSummary.ConsumedAmount,
				ConsumptionRatio: parsed.UsageSummary.ConsumptionRatio,
			}
			if summary.TotalAmount == 0 && parsed.Data.UsageSummary.TotalAmount > 0 {
				summary.TotalAmount = parsed.Data.UsageSummary.TotalAmount
				summary.ConsumedAmount = parsed.Data.UsageSummary.ConsumedAmount
				summary.ConsumptionRatio = parsed.Data.UsageSummary.ConsumptionRatio
			}
			summary.RemainingAmount = math.Max(0, summary.TotalAmount-summary.ConsumedAmount)

			rawPacks := parsed.UserEntitlementPackList
			if len(rawPacks) == 0 {
				rawPacks = parsed.Data.UserEntitlementPackList
			}

			var packs []EntitlementPack
			for _, item := range rawPacks {
				if item.EntitlementBaseInfo.ProductType == 3 {
					continue
				}

				name := item.GroupName
				if name == "" {
					name = item.DisplayDesc
				}
				if name == "" {
					name = item.PackName
				}
				if name == "" {
					name = "通用权益包"
				}

				total := item.EntitlementBaseInfo.Quota.CreditsLimit
				if total == 0 {
					total = item.EntitlementBaseInfo.Quota.TotalQuota
				}
				if total == 0 {
					total = item.CreditsLimit
				}
				if total == 0 {
					total = item.TotalQuota
				}
				if total == 0 {
					total = item.Quota
				}

				used := item.Usage.CreditsAmount
				if used == 0 {
					used = item.Usage.UsedQuota
				}

				unit := item.EntitlementBaseInfo.Quota.Unit
				if unit == "" {
					unit = "积分"
				}

				category := "general"
				if item.EntitlementBaseInfo.AvailableEndpoint == 1 || item.EntitlementBaseInfo.ProductID == 209 {
					category = "work"
				} else {
					descLow := strings.ToLower(name + " " + item.PackDesc + " " + item.GroupName + " " + item.DisplayDesc)
					if strings.Contains(descLow, "work") {
						category = "work"
					}
				}

				packs = append(packs, EntitlementPack{
					PackID:     item.PackID,
					PackName:   name,
					PackDesc:   item.PackDesc,
					Category:   category,
					TotalQuota: total,
					UsedQuota:  used,
					Unit:       unit,
					ExpireTime: item.ExpireTime,
					Status:     item.Status,
				})
			}

			return &GetEntitlementsResult{
				Summary: summary,
				Packs:   packs,
			}, nil
		}
	}

	return &GetEntitlementsResult{}, nil
}

// GetCheckinStatus retrieves credits balance and today's checkin status
func (c *Client) GetCheckinStatus(ctx context.Context, token string, deviceID string) (*CheckinInfo, error) {
	headers := map[string]string{
		"Authorization": "Cloud-IDE-JWT " + token,
		"x-app-type":     "trae",
		"Origin":         "https://www.trae.cn",
		"Referer":        "https://www.trae.cn/",
	}
	reqPath := PathCheckinStatus
	if deviceID != "" {
		headers["x-device-id"] = deviceID
		reqPath += "?did=" + url.QueryEscape(deviceID)
	}

	raw, err := c.doRequest(ctx, http.MethodGet, reqPath, headers, nil)
	if err != nil {
		return nil, err
	}

	var parsed struct {
		Code         int     `json:"code"`
		CheckedIn    bool    `json:"checked_in"`
		CanCheckin   bool    `json:"can_checkin"`
		Credits      float64 `json:"credits"`
		ExtraCredits float64 `json:"extra_credits"`
		Enable       bool    `json:"enable"`
		Data         struct {
			CheckedIn    bool    `json:"checked_in"`
			CanCheckin   bool    `json:"can_checkin"`
			Credits      float64 `json:"credits"`
			ExtraCredits float64 `json:"extra_credits"`
			Enable       bool    `json:"enable"`
		} `json:"data"`
	}

	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}

	checkedIn := parsed.CheckedIn || parsed.Data.CheckedIn
	credits := parsed.Credits
	if credits == 0 {
		credits = parsed.Data.Credits
	}
	extra := parsed.ExtraCredits
	if extra == 0 {
		extra = parsed.Data.ExtraCredits
	}
	enable := parsed.Enable || parsed.Data.Enable

	return &CheckinInfo{
		CheckedIn:    checkedIn,
		CanCheckin:   enable && !checkedIn,
		Credits:      credits,
		ExtraCredits: extra,
	}, nil
}

// ClaimCheckin performs the daily checkin to claim free credits
func (c *Client) ClaimCheckin(ctx context.Context, token string, deviceID string) (*CheckinResult, error) {
	headers := map[string]string{
		"Authorization": "Cloud-IDE-JWT " + token,
		"x-app-type":     "trae",
		"Origin":         "https://www.trae.cn",
		"Referer":        "https://www.trae.cn/",
	}
	if deviceID != "" {
		headers["x-device-id"] = deviceID
	}

	raw, err := c.doRequest(ctx, http.MethodPost, PathCheckinClaim, headers, map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	var parsed struct {
		Code          int     `json:"code"`
		Message       string  `json:"message"`
		CreditsEarned float64 `json:"credits_earned"`
		Credits       float64 `json:"credits"`
		Data          struct {
			CreditsEarned float64 `json:"credits_earned"`
			NewBalance    float64 `json:"credits"`
		} `json:"data"`
	}

	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}

	earned := parsed.CreditsEarned
	if earned == 0 {
		earned = parsed.Data.CreditsEarned
	}
	balance := parsed.Credits
	if balance == 0 {
		balance = parsed.Data.NewBalance
	}

	if parsed.Code == 0 || parsed.Code == 200 {
		return &CheckinResult{
			Success:       true,
			CreditsEarned: earned,
			NewBalance:    balance,
			Message:       "签到成功！",
		}, nil
	}

	already := strings.Contains(parsed.Message, "已经") || strings.Contains(parsed.Message, "重复")
	return &CheckinResult{
		Success:        false,
		AlreadyClaimed: already,
		Message:        parsed.Message,
	}, nil
}

// GetUsageRecords retrieves recent session billing records from Trae cloud
func (c *Client) GetUsageRecords(ctx context.Context, token string, page, pageSize int) ([]UsageSessionRecord, int, error) {
	headers := map[string]string{
		"Authorization": "Cloud-IDE-JWT " + token,
	}

	if pageSize <= 0 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}

	nowSec := time.Now().Unix()
	startTime := nowSec - 30*24*3600 // 最近 30 天
	endTime := nowSec

	baseBody := map[string]interface{}{
		"start_time": startTime,
		"end_time":   endTime,
		"page_num":   page,
		"page_size":  pageSize,
		"Request":    map[string]interface{}{},
	}

	// 尝试两种方案：优先 usage_type: [7] (积分计费模式)，若无则尝试不带 usage_type
	attempts := []map[string]interface{}{
		{
			"start_time": startTime,
			"end_time":   endTime,
			"page_num":   page,
			"page_size":  pageSize,
			"usage_type": []int{7},
			"Request":    map[string]interface{}{},
		},
		baseBody,
	}

	type sessionItem struct {
		SessionID        string  `json:"session_id"`
		StartTime        int64   `json:"session_start_time"`
		EndTime          int64   `json:"session_end_time"`
		ModelName        string  `json:"model_name"`
		CreditsConsumed  float64 `json:"credits_float"`
		PromptTokens     int64   `json:"prompt_tokens"`
		CompletionTokens int64   `json:"completion_tokens"`
		UserInputPreview string  `json:"user_input_preview"`
		ProductTypeList  []int   `json:"product_type_list"`
	}

	for _, body := range attempts {
		raw, err := c.doRequest(ctx, http.MethodPost, PathUsageGroupBySession, headers, body)
		if err != nil {
			continue
		}

		var parsed struct {
			Code     int           `json:"code"`
			Total    int           `json:"total"`
			Sessions []sessionItem `json:"user_usage_group_by_sessions"`
			Data     struct {
				Total    int           `json:"total"`
				Sessions []sessionItem `json:"user_usage_group_by_sessions"`
			} `json:"data"`
			Result struct {
				Total    int           `json:"total"`
				Sessions []sessionItem `json:"user_usage_group_by_sessions"`
			} `json:"result"`
		}

		if err := json.Unmarshal(raw, &parsed); err == nil {
			total := parsed.Total
			sessions := parsed.Sessions
			if total == 0 && len(sessions) == 0 {
				total = parsed.Data.Total
				sessions = parsed.Data.Sessions
			}
			if total == 0 && len(sessions) == 0 {
				total = parsed.Result.Total
				sessions = parsed.Result.Sessions
			}

			if len(sessions) > 0 {
				var records []UsageSessionRecord
				for _, s := range sessions {
					product := "Free"
					if len(s.ProductTypeList) > 0 {
						switch s.ProductTypeList[0] {
						case 1:
							product = "Pro"
						case 2:
							product = "Package"
						case 4:
							product = "ProPlus"
						case 6:
							product = "Ultra"
						case 100:
							product = "Express"
						}
					}

					records = append(records, UsageSessionRecord{
						SessionID:        s.SessionID,
						StartTime:        s.StartTime,
						EndTime:          s.EndTime,
						ModelName:        s.ModelName,
						ProductName:      product,
						CreditsConsumed:  s.CreditsConsumed,
						PromptTokens:     s.PromptTokens,
						CompletionTokens: s.CompletionTokens,
						TotalTokens:      s.PromptTokens + s.CompletionTokens,
						Preview:          s.UserInputPreview,
					})
				}
				return records, total, nil
			}
		}
	}

	return []UsageSessionRecord{}, 0, nil
}

// GetFullProfile queries all profile data in parallel and calculates spendable credits
func (c *Client) GetFullProfile(ctx context.Context, token string, deviceID string) (*ProfileResponse, error) {
	resp := &ProfileResponse{}

	// UserInfo
	if u, err := c.GetUserInfo(ctx, token); err == nil && u != nil {
		resp.UserInfo = *u
	}
	// PayStatus
	if p, err := c.GetPayStatus(ctx, token); err == nil && p != nil {
		resp.PayStatus = *p
	}
	// Entitlements & UsageSummary
	if entRes, err := c.GetEntitlements(ctx, token); err == nil && entRes != nil {
		resp.Entitlements = entRes.Packs
		resp.UsageSummary = entRes.Summary
	}
	// Checkin & Credits
	if ch, err := c.GetCheckinStatus(ctx, token, deviceID); err == nil && ch != nil {
		resp.Checkin = *ch
	}

	// Calculate Spendable, General, and Work Credits
	var workCredits float64
	for _, pack := range resp.Entitlements {
		rem := math.Max(0, pack.TotalQuota-pack.UsedQuota)
		if pack.Category == "work" {
			workCredits += rem
		}
	}

	if resp.UsageSummary.TotalAmount > 0 {
		resp.TotalCredits = resp.UsageSummary.RemainingAmount
	} else if len(resp.Entitlements) > 0 {
		var sum float64
		for _, pack := range resp.Entitlements {
			sum += math.Max(0, pack.TotalQuota-pack.UsedQuota)
		}
		resp.TotalCredits = sum
	} else {
		resp.TotalCredits = resp.Checkin.Credits + resp.Checkin.ExtraCredits
	}

	resp.WorkCredits = workCredits
	resp.GeneralCredits = math.Max(0, resp.TotalCredits-workCredits)
	resp.SpendableCredits = resp.GeneralCredits
	if resp.SpendableCredits == 0 && resp.TotalCredits > 0 {
		resp.SpendableCredits = resp.TotalCredits
	}

	return resp, nil
}
