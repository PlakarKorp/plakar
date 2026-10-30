package api

import "net/http"

// With -no-auth, any page could POST here: refuse cross-site and same-site
// requests, a page on another local port is same-site.
func (ui *uiserver) quit(w http.ResponseWriter, r *http.Request) error {
	if site := r.Header.Get("Sec-Fetch-Site"); site == "cross-site" || site == "same-site" {
		return &ApiError{
			HttpCode: http.StatusForbidden,
			ErrCode:  "forbidden",
			Message:  "quit is refused to other sites",
		}
	}
	ui.ctx.Cancel(ErrQuit)
	return nil
}
