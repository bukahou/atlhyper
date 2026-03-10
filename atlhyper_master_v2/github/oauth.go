// atlhyper_master_v2/github/oauth.go
// OAuth 流程实现 — 实现 auth.OAuthProvider
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"AtlHyper/atlhyper_master_v2/auth"
	"AtlHyper/atlhyper_master_v2/database"
)

// AuthURL 生成 GitHub OAuth 授权 URL
func (c *clientImpl) AuthURL(state string) string {
	params := url.Values{
		"client_id": {c.cfg.ClientID},
		"state":     {state},
	}
	return "https://github.com/login/oauth/authorize?" + params.Encode()
}

// Exchange 用授权码换取用户信息
func (c *clientImpl) Exchange(ctx context.Context, code string) (*auth.OAuthUser, error) {
	// 1. 用 code 换 access token
	tokenURL := "https://github.com/login/oauth/access_token"
	data := url.Values{
		"client_id":     {c.cfg.ClientID},
		"client_secret": {c.cfg.ClientSecret},
		"code":          {code},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}
	defer resp.Body.Close()

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}
	if tokenResp.Error != "" {
		return nil, fmt.Errorf("OAuth error: %s", tokenResp.Error)
	}

	// 2. 用 access token 获取用户信息
	userReq, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	userReq.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
	userReq.Header.Set("Accept", "application/vnd.github+json")

	userResp, err := c.httpClient.Do(userReq)
	if err != nil {
		return nil, fmt.Errorf("get user info failed: %w", err)
	}
	defer userResp.Body.Close()

	if userResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(userResp.Body)
		return nil, fmt.Errorf("get user info failed: %d %s", userResp.StatusCode, string(body))
	}

	var ghUser struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.NewDecoder(userResp.Body).Decode(&ghUser); err != nil {
		return nil, err
	}

	// 3. 获取用户的 App installations
	installReq, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/installations", nil)
	if err != nil {
		return nil, err
	}
	installReq.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
	installReq.Header.Set("Accept", "application/vnd.github+json")

	installResp, err := c.httpClient.Do(installReq)
	if err != nil {
		log.Warn("获取 installations 失败", "err", err)
	} else {
		defer installResp.Body.Close()
		if installResp.StatusCode == http.StatusOK {
			var installResult struct {
				Installations []struct {
					ID      int64 `json:"id"`
					AppID   int64 `json:"app_id"`
					Account struct {
						Login string `json:"login"`
					} `json:"account"`
				} `json:"installations"`
			}
			if err := json.NewDecoder(installResp.Body).Decode(&installResult); err == nil {
				for _, inst := range installResult.Installations {
					if inst.AppID == c.cfg.AppID {
						c.SetInstallationID(inst.ID)
						log.Info("找到 App Installation", "id", inst.ID, "account", inst.Account.Login)
						break
					}
				}
			}
		}
	}

	return &auth.OAuthUser{
		Provider:   "github",
		ExternalID: fmt.Sprintf("%d", ghUser.ID),
		Login:      ghUser.Login,
		Email:      ghUser.Email,
		AvatarURL:  ghUser.AvatarURL,
	}, nil
}

// ExchangeForConnection 用授权码完成连接，返回连接状态和安装记录
// 供 Handler 层直接调用，避免 Handler 需要了解 auth.OAuthUser 的细节
func (c *clientImpl) ExchangeForConnection(ctx context.Context, code string) (*ConnectionStatus, *database.GitHubInstallation, error) {
	user, err := c.Exchange(ctx, code)
	if err != nil {
		return nil, nil, err
	}

	c.mu.RLock()
	installID := c.installationID
	c.mu.RUnlock()

	status := &ConnectionStatus{
		Connected:      installID > 0,
		AccountLogin:   user.Login,
		AvatarURL:      user.AvatarURL,
		InstallationID: installID,
	}

	var inst *database.GitHubInstallation
	if installID > 0 {
		inst = &database.GitHubInstallation{
			InstallationID: installID,
			AccountLogin:   user.Login,
		}
	}

	return status, inst, nil
}
