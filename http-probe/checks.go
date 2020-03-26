package http-probe

import (
	"fmt"
	"net"
	"net/http"
	"time"
)



// GetCheck returns a Check that performs an  GET request against the
// specified URL. The check fails if the response times out or returns a non-200
// status code
func GetCheck(url string, timeout time.Duration) error {
	client := http.Client{
		Timeout: timeout,
		// never follow redirects
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Get(url)
		if err != nil {
			return err
		}
		resp.Body.Close()
		if resp.StatusCode != 200 {
			return fmt.Errorf("http status code  %d", resp.StatusCode)
		}
		return nil
}