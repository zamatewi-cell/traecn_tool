package traeapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	PlanType      string `json:"plan_type"` // e.g. Free, Pro, Ultra, Express
	ExpireAt      int64  `json:"expire_at,omitempty"`
	IsPayFreshman bool   `json:"is_pay_freshman"`
}

// EntitlementPack represents a quota package (e.g. 500 requests/month)
type EntitlementPack struct {
	PackID     string `json:"pack_id"`
	PackName   string `json:"pack_name"`
	PackDesc   string `json:"pack_desc"`
	TotalQuota int64  `json:"total_quota"`
	UsedQuota  int64  `json:"used_quota"`
	Unit       string `json:"unit"`
	ExpireTime int64  `json:"expire_time"`
	Status     int    `json:"status"`
}

// CheckinInfo represents daily checkin status & credits balance
type CheckinInfo struct {
	CheckedIn  bool  `json:"checked_in"`
	CanCheckin bool  `json:"can_checkin"`
	Credits    int64 `json:"credits"`
}

// CheckinResult is the return value of ClaimCheckin
type CheckinResult struct {
	Success        bool   `json:"success"`
	AlreadyClaimed bool   `json:"already_claimed"`
	CreditsEarned  int64  `json:"credits_earned"`
	NewBalance     int64  `json:"new_balance"`
	Message        string `json:"message"`
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
	UserInfo     UserInfo          `json:"user_info"`
	PayStatus    PayStatus         `json:"pay_status"`
	Entitlements []EntitlementPack `json:"entitlements"`
	Checkin      CheckinInfo       `json:"checkin"`
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
			Code int `json:"code"`
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
			plan := parsed.Data.UserPayIdentityStr
			renew := parsed.Data.Detail.SubscriptionRenewTime
			freshman := parsed.Data.IsPayFreshman
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

// GetEntitlements retrieves quota package usage (500/month, etc.)
func (c *Client) GetEntitlements(ctx context.Context, token string) ([]EntitlementPack, error) {
	headers := map[string]string{
		"Authorization": "Cloud-IDE-JWT " + token,
	}

	paths := []string{PathEntUsageV2, PathEntUsageV1}
	for _, p := range paths {
		raw, err := c.doRequest(ctx, http.MethodPost, p, headers, map[string]interface{}{"require_usage": true})
		if err != nil {
			continue
		}

		var parsed struct {
			Code int `json:"code"`
			Data struct {
				UserEntitlementPackList []struct {
					PackID              string `json:"entitlement_pack_id"`
					PackName            string `json:"entitlement_pack_name"`
					PackDesc            string `json:"entitlement_pack_desc"`
					Status              int    `json:"status"`
					ExpireTime          int64  `json:"expire_time"`
					EntitlementBaseInfo struct {
						ProductType int `json:"product_type"`
						Quota       struct {
							TotalQuota int64  `json:"total_quota"`
							Unit       string `json:"unit"`
						} `json:"quota"`
					} `json:"entitlement_base_info"`
					Usage struct {
						CreditsAmount int64 `json:"credits_amount"`
					} `json:"usage"`
				} `json:"user_entitlement_pack_list"`
			} `json:"data"`
		}

		if err := json.Unmarshal(raw, &parsed); err == nil && len(parsed.Data.UserEntitlementPackList) > 0 {
			var packs []EntitlementPack
			for _, item := range parsed.Data.UserEntitlementPackList {
				// Filter out promo product_type == 3
				if item.EntitlementBaseInfo.ProductType == 3 {
					continue
				}
				name := item.PackName
				if name == "" {
					name = "通用权益包"
				}
				unit := item.EntitlementBaseInfo.Quota.Unit
				if unit == "" {
					unit = "次"
				}
				packs = append(packs, EntitlementPack{
					PackID:     item.PackID,
					PackName:   name,
					PackDesc:   item.PackDesc,
					TotalQuota: item.EntitlementBaseInfo.Quota.TotalQuota,
					UsedQuota:  item.Usage.CreditsAmount,
					Unit:       unit,
					ExpireTime: item.ExpireTime,
					Status:     item.Status,
				})
			}
			return packs, nil
		}
	}

	return []EntitlementPack{}, nil
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
		Code int `json:"code"`
		Data struct {
			CheckedIn  bool  `json:"checked_in"`
			CanCheckin bool  `json:"can_checkin"`
			Credits    int64 `json:"credits"`
			Enable     bool  `json:"enable"`
		} `json:"data"`
	}

	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}

	return &CheckinInfo{
		CheckedIn:  parsed.Data.CheckedIn,
		CanCheckin: parsed.Data.Enable && !parsed.Data.CheckedIn,
		Credits:    parsed.Data.Credits,
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
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			CreditsEarned int64 `json:"credits_earned"`
			NewBalance    int64 `json:"credits"`
		} `json:"data"`
	}

	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}

	if parsed.Code == 0 || parsed.Code == 200 {
		return &CheckinResult{
			Success:       true,
			CreditsEarned: parsed.Data.CreditsEarned,
			NewBalance:    parsed.Data.NewBalance,
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

	body := map[string]interface{}{
		"page_num":   page,
		"page_size":  pageSize,
		"usage_type": []int{1, 2},
	}

	raw, err := c.doRequest(ctx, http.MethodPost, PathUsageGroupBySession, headers, body)
	if err != nil {
		return nil, 0, err
	}

	var parsed struct {
		Code int `json:"code"`
		Data struct {
			Total    int `json:"total"`
			Sessions []struct {
				SessionID        string  `json:"session_id"`
				StartTime        int64   `json:"session_start_time"`
				EndTime          int64   `json:"session_end_time"`
				ModelName        string  `json:"model_name"`
				CreditsConsumed  float64 `json:"credits_float"`
				PromptTokens     int64   `json:"prompt_tokens"`
				CompletionTokens int64   `json:"completion_tokens"`
				UserInputPreview string  `json:"user_input_preview"`
				ProductTypeList  []int   `json:"product_type_list"`
			} `json:"user_usage_group_by_sessions"`
		} `json:"data"`
	}

	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, 0, err
	}

	var records []UsageSessionRecord
	for _, s := range parsed.Data.Sessions {
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

	return records, parsed.Data.Total, nil
}

// GetFullProfile queries all profile data in parallel
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
	// Entitlements
	if e, err := c.GetEntitlements(ctx, token); err == nil && e != nil {
		resp.Entitlements = e
	}
	// Checkin & Credits
	if ch, err := c.GetCheckinStatus(ctx, token, deviceID); err == nil && ch != nil {
		resp.Checkin = *ch
	}

	return resp, nil
}
