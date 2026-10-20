package api

import "net/http"

// With -no-auth, any page could POST here: refuse cross-site requests.
func (ui *uiserver) quit(w http.ResponseWriter, r *http.Request) error {
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		return &ApiError{
			HttpCode: http.StatusForbidden,
			ErrCode:  "forbidden",
			Message:  "quit is refused to other sites",
		}
	}
	ui.ctx.Cancel(ErrQuit)
	return nil
}
