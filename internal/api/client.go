package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// RequestTimeout caps every API call. Without it a stalled connection hangs the whole
// run forever (a scheduled job would then never notify or exit — observed in testing).
const RequestTimeout = 15 * time.Second

type Client struct {
	base string
	http *http.Client
}

func New(base string, hc *http.Client) *Client {
	if hc == nil {
		hc = &http.Client{Timeout: RequestTimeout}
	}
	return &Client{base: base, http: hc}
}

func (c *Client) headers(req *http.Request) {
	req.Header.Set("x-client-type", "h5")
	req.Header.Set("x-catering-timezone", "GMT+8:00")
	req.Header.Set("x-accept-language", "en")
	req.Header.Set("content-type", "application/json; charset=UTF-8")
	req.Header.Set("accept", "application/json, text/plain, */*")
}

func (c *Client) do(req *http.Request, out any) error {
	c.headers(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) Menu(building, date, timeCode string) (*MenuResp, error) {
	u := fmt.Sprintf("%s/inner-order/menu/v3?buildingCode=%s&mealDate=%s&timeCode=%s&isFilterEmptyStations=1",
		c.base, url.QueryEscape(building), date, timeCode)
	req, _ := http.NewRequest("GET", u, nil)
	var out MenuResp
	return &out, c.do(req, &out)
}

func (c *Client) Calendar(building, timeCode string) (*CalendarResp, error) {
	body, _ := json.Marshal(map[string]string{"buildingCode": building, "timeCode": timeCode})
	req, _ := http.NewRequest("POST", c.base+"/inner-order/calendar", bytes.NewReader(body))
	var out CalendarResp
	return &out, c.do(req, &out)
}

func (c *Client) Submit(r SubmitReq) (*SubmitResp, error) {
	body, _ := json.Marshal(r)
	req, _ := http.NewRequest("POST", c.base+"/inner-order/submit-order/batch", bytes.NewReader(body))
	var out SubmitResp
	return &out, c.do(req, &out)
}

func (c *Client) UserInfo() (map[string]any, error) {
	req, _ := http.NewRequest("GET", c.base+"/mini-program/h5/user_info", nil)
	out := map[string]any{}
	return out, c.do(req, &out)
}
