package jobcan

import (
	"bytes"
	"context"
	"fmt"
	"regexp"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
	cookiejar "github.com/juju/persistent-cookiejar"
)

type Client struct {
	username string
	password string
	client   *resty.Client
	cjar     *Jar
}

func NewClient(cookiefile, username, password string, debug bool) *Client {
	cjar := NewJar(&cookiejar.Options{Filename: cookiefile})
	client := resty.New().SetCookieJar(cjar).
		SetHeader("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/94.0.4606.61 Safari/537.36").
		SetHeader("sec-ch-ua", `"Chromium";v="94", "Google Chrome";v="94", ";Not A Brand";v="99"`).
		SetHeader("origin", "https://id.jobcan.jp").
		SetHeader("accept", "text/html").
		SetHeader("sec-ch-ua", `"Chromium";v="94", "Google Chrome";v="94", ";Not A Brand";v="99"`).
		SetHeader("referer", "https://id.jobcan.jp/").
		SetDebug(debug)
	return &Client{
		username: username,
		password: password,
		client:   client,
		cjar:     cjar,
	}
}

const (
	loginURL  = "https://id.jobcan.jp/users/sign_in?app_key=atd&redirect_to=https://ssl.jobcan.jp/jbcoauth/callback"
	signinURL = "https://id.jobcan.jp/users/sign_in"
	statusURL = "https://ssl.jobcan.jp/employee"
	editURL   = "https://ssl.jobcan.jp/employee/index/adit"
)

var (

	// cjar   *cookiejar.Jar
	expr = regexp.MustCompile(`var current_status = "(.*?)";`)
)

func init() {

}

func loadFromByte(in []byte) *goquery.Document {
	buf := bytes.NewBuffer(in)
	doc, err := goquery.NewDocumentFromReader(buf)
	if err != nil {
		panic(err)
	}
	return doc
}

func (c *Client) Login(ctx context.Context) (status string, token string, err error) {
	resp, err := c.client.R().SetContext(ctx).Get(loginURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to access login page: %w", err)
	} else if resp.StatusCode() != 200 {
		return "", "", fmt.Errorf("http failed: %s", resp.Status())
	}

	doc := loadFromByte(resp.Body())
	csrfToken, ok := doc.Find(`meta[name="csrf-token"]`).Attr("content")
	if !ok {
		return "", "", fmt.Errorf("unable to get csrf-token")
	}

	resp, err = c.client.R().SetContext(ctx).
		SetFormData(map[string]string{
			"authenticity_token": csrfToken,
			"user[email]":        c.username,
			"user[client_code]":  "",
			"user[password]":     c.password,
			"app_key":            "atd",
			"commit":             "Login",
		}).Post(signinURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to login: %w", err)
	} else if resp.StatusCode() != 200 {
		return "", "", fmt.Errorf("http failed: %s", resp.Status())
	}
	doc = loadFromByte(resp.Body())
	token, _ = doc.Find(`input[name="token"]`).Attr("value")
	status, err = extractStatus(resp.Body())
	return status, token, err
}

func (c *Client) GetStatus(ctx context.Context) (status string, err error) {
	if !c.IsLogined() {
		status, _, err = c.Login(ctx)
		return
	}
	resp, err := c.client.R().SetContext(ctx).Get(statusURL)
	if err != nil {
		return "", fmt.Errorf("failed to check status: %w", err)
	} else if resp.StatusCode() != 200 {
		return "", fmt.Errorf("http failed: %s", resp.Status())
	}
	return extractStatus(resp.Body())
}

func (c *Client) Toggle(ctx context.Context) (prevStatus string, newStatus string, err error) {
	var token string
	type resultJSON struct {
		CurrentStatus string `json:"current_status"`
		Result        int    `json:"result"`
		State         int    `json:"state"`
	}

	if !c.IsLogined() {
		prevStatus, token, err = c.Login(ctx)
		if err != nil {
			return "", "", err
		}

	} else {
		var ok bool
		resp, err := c.client.R().SetContext(ctx).Get(statusURL)
		if err != nil {
			return "", "", fmt.Errorf("failed to check status: %w", err)
		}
		prevStatus, err = extractStatus(resp.Body())
		if err != nil {
			return "", "", err
		}
		doc := loadFromByte(resp.Body())
		token, ok = doc.Find(`input[name="token"]`).Attr("value")
		if !ok {
			return "", "", fmt.Errorf("unable to get token")
		} else if resp.StatusCode() != 200 {
			return "", "", fmt.Errorf("http failed: %s", resp.Status())
		}
	}

	j := &resultJSON{}
	resp, err := c.client.R().SetContext(ctx).SetFormData(map[string]string{
		"is_yakin":      "0",
		"adit_item":     "DEF",
		"notice":        "",
		"token":         token,
		"adit_group_id": "3",
		"_":             "",
	}).SetResult(j).
		Post(editURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to clock off: %w", err)
	} else if resp.StatusCode() != 200 {
		return "", "", fmt.Errorf("http failed: %s", resp.Status())
	}
	return prevStatus, j.CurrentStatus, nil
}

func (c *Client) Reset() {
	c.cjar.RemoveAll()
}

func (c *Client) Close() error {
	return c.cjar.Save()
}

func (c *Client) IsLogined() bool {
	return !c.cjar.CheckSession()
}

func extractStatus(in []byte) (string, error) {
	out := expr.FindStringSubmatch(string(in))
	if len(out) != 2 {
		return "", fmt.Errorf("unable to extract working status: got %v", out)
	}
	return out[1], nil
}
