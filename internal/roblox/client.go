package roblox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	apiBaseURL      = "https://users.roblox.com"
	gamesBaseURL    = "https://games.roblox.com"
	friendsBaseURL  = "https://friends.roblox.com"
	presenceBaseURL = "https://presence.roblox.com"
	authBaseURL     = "https://auth.roblox.com"
	maxResponseSize = 4 << 20
)

type Client struct {
	httpClient      *http.Client
	apiBaseURL      string
	gamesBaseURL    string
	friendsBaseURL  string
	presenceBaseURL string
	authBaseURL     string
}

type User struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

type Favorite struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	RootPlaceID int64  `json:"rootPlaceId"`
}

type Friend struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

type FriendPresence struct {
	Friend
	State        string
	LastLocation string
}

type friendListResponse struct {
	Data []Friend `json:"data"`
}

type favoriteListResponse struct {
	Data []Favorite `json:"data"`
}

type presenceListResponse struct {
	UserPresences []presenceResponse `json:"userPresences"`
}

type presenceResponse struct {
	UserID       int64  `json:"userId"`
	PresenceType int    `json:"userPresenceType"`
	LastLocation string `json:"lastLocation"`
}

func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}
	return &Client{
		httpClient:      httpClient,
		apiBaseURL:      apiBaseURL,
		gamesBaseURL:    gamesBaseURL,
		friendsBaseURL:  friendsBaseURL,
		presenceBaseURL: presenceBaseURL,
		authBaseURL:     authBaseURL,
	}
}

func (c *Client) AuthenticatedUser(ctx context.Context, cookie string) (User, error) {
	cleanCookie, err := normalizeCookie(cookie)
	if err != nil {
		return User{}, err
	}
	var user User
	if err := c.getJSON(ctx, c.apiBaseURL+"/v1/users/authenticated", cleanCookie, &user); err != nil {
		return User{}, fmt.Errorf("fetch authenticated Roblox user: %w", err)
	}
	if user.ID <= 0 {
		return User{}, errors.New("Roblox returned an invalid authenticated user")
	}
	return user, nil
}

func (c *Client) FavoriteGames(ctx context.Context, cookie string, userID int64) ([]Favorite, error) {
	if userID <= 0 {
		return nil, errors.New("user ID must be positive")
	}
	cleanCookie, err := normalizeCookie(cookie)
	if err != nil {
		return nil, err
	}
	endpoint := c.gamesBaseURL + "/v2/users/" + strconv.FormatInt(userID, 10) + "/favorite/games?limit=50&sortOrder=Desc"
	var result favoriteListResponse
	if err := c.getJSON(ctx, endpoint, cleanCookie, &result); err != nil {
		return nil, fmt.Errorf("fetch favorite games: %w", err)
	}
	return result.Data, nil
}

func (c *Client) FriendsWithPresence(ctx context.Context, cookie string) ([]FriendPresence, error) {
	cleanCookie, err := normalizeCookie(cookie)
	if err != nil {
		return nil, err
	}

	var result friendListResponse
	if err := c.getJSON(ctx, c.friendsBaseURL+"/v1/my/friends?userSort=Asc", cleanCookie, &result); err != nil {
		return nil, fmt.Errorf("fetch Roblox friends: %w", err)
	}
	friends := make([]FriendPresence, 0, len(result.Data))
	for _, friend := range result.Data {
		friends = append(friends, FriendPresence{Friend: friend, State: "Unknown"})
	}

	for start := 0; start < len(result.Data); start += 100 {
		end := min(start+100, len(result.Data))
		ids := make([]int64, 0, end-start)
		for _, friend := range result.Data[start:end] {
			ids = append(ids, friend.ID)
		}
		body, err := json.Marshal(struct {
			UserIDs []int64 `json:"userIds"`
		}{UserIDs: ids})
		if err != nil {
			return nil, fmt.Errorf("encode presence request: %w", err)
		}
		presences, err := c.fetchPresences(ctx, cleanCookie, body)
		if err != nil {
			return nil, fmt.Errorf("fetch friend presence: %w", err)
		}
		for _, presence := range presences {
			index := friendIndex(result.Data, presence.UserID)
			if index < 0 {
				continue
			}
			friends[index].State = presenceName(presence.PresenceType)
			friends[index].LastLocation = presence.LastLocation
		}
	}
	return friends, nil
}

func (c *Client) AuthenticationTicket(ctx context.Context, cookie string) (string, error) {
	cleanCookie, err := normalizeCookie(cookie)
	if err != nil {
		return "", err
	}

	var csrfToken string
	for attempt := 0; attempt < 2; attempt++ {
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.authBaseURL+"/v1/authentication-ticket", nil)
		if err != nil {
			return "", fmt.Errorf("create authentication-ticket request: %w", err)
		}
		request.Header.Set("Cookie", ".ROBLOSECURITY="+cleanCookie)
		request.Header.Set("Origin", "https://www.roblox.com")
		request.Header.Set("Referer", "https://www.roblox.com/")
		request.Header.Set("User-Agent", "ShotGlass/1.0")
		if csrfToken != "" {
			request.Header.Set("X-CSRF-TOKEN", csrfToken)
		}

		response, err := c.httpClient.Do(request)
		if err != nil {
			return "", fmt.Errorf("request Roblox authentication ticket: %w", err)
		}
		_, readErr := io.Copy(io.Discard, io.LimitReader(response.Body, maxResponseSize))
		response.Body.Close()
		if readErr != nil {
			return "", fmt.Errorf("read Roblox authentication-ticket response: %w", readErr)
		}
		if response.StatusCode == http.StatusForbidden && attempt == 0 && response.Header.Get("X-CSRF-TOKEN") != "" {
			csrfToken = response.Header.Get("X-CSRF-TOKEN")
			continue
		}
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			return "", fmt.Errorf("Roblox authentication-ticket endpoint returned HTTP %d", response.StatusCode)
		}
		ticket := response.Header.Get("rbx-authentication-ticket")
		if ticket == "" {
			return "", errors.New("Roblox authentication-ticket response did not include a ticket")
		}
		return ticket, nil
	}
	return "", errors.New("Roblox rejected the authentication-ticket request")
}

func (c *Client) getJSON(ctx context.Context, endpoint, cookie string, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("create Roblox API request: %w", err)
	}
	request.Header.Set("Cookie", ".ROBLOSECURITY="+cookie)
	request.Header.Set("User-Agent", "ShotGlass/1.0")
	return c.doJSON(request, target)
}

func (c *Client) fetchPresences(ctx context.Context, cookie string, body []byte) ([]presenceResponse, error) {
	endpoint := c.presenceBaseURL + "/v1/presence/users"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create Roblox presence request: %w", err)
	}
	request.Header.Set("Cookie", ".ROBLOSECURITY="+cookie)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "ShotGlass/1.0")
	var result presenceListResponse
	if err := c.doJSON(request, &result); err != nil {
		return nil, err
	}
	return result.UserPresences, nil
}

func (c *Client) doJSON(request *http.Request, target any) error {
	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("request Roblox API: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Roblox API %s returned HTTP %d", request.URL.Host+request.URL.Path, response.StatusCode)
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, maxResponseSize))
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode Roblox API response: %w", err)
	}
	return nil
}

func normalizeCookie(cookie string) (string, error) {
	cookie = strings.TrimSpace(cookie)
	cookie = strings.TrimPrefix(cookie, ".ROBLOSECURITY=")
	if cookie == "" {
		return "", errors.New("enter a Roblox .ROBLOSECURITY cookie")
	}
	for _, value := range []byte(cookie) {
		if !isCookieOctet(value) {
			return "", errors.New("the Roblox cookie contains invalid characters")
		}
	}
	return cookie, nil
}

func isCookieOctet(value byte) bool {
	return value == 0x21 ||
		value >= 0x23 && value <= 0x2b ||
		value >= 0x2d && value <= 0x3a ||
		value >= 0x3c && value <= 0x5b ||
		value >= 0x5d && value <= 0x7e
}

func friendIndex(friends []Friend, userID int64) int {
	for i, friend := range friends {
		if friend.ID == userID {
			return i
		}
	}
	return -1
}

func presenceName(presenceType int) string {
	switch presenceType {
	case 0:
		return "Offline"
	case 1:
		return "Online"
	case 2:
		return "In game"
	case 3:
		return "In Studio"
	default:
		return "Unknown"
	}
}
