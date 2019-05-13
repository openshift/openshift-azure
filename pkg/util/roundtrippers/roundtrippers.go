package roundtrippers

import (
	"net/http"
)

// AuthorizingRoundTripper acts as a normal RoundTripper, but with a bearer token
type AuthorizingRoundTripper struct {
	http.RoundTripper
	Token string
}

func (rt AuthorizingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+rt.Token)
	return rt.RoundTripper.RoundTrip(req)
}
