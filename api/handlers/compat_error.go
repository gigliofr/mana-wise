package handlers

import "net/http"

// jsonError is kept as a compatibility shim during migration. It delegates
// to WriteAPIErrorFromMsg which implements the standardized response shape.
func jsonError(w http.ResponseWriter, msg string, code int) {
    WriteAPIErrorFromMsg(w, msg, code)
}
