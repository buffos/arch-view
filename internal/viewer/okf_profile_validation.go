package viewer

import (
	"context"
	"net/http"
)

func (s *Server) handleOKFValidateProfile(writer http.ResponseWriter, request *http.Request, ctx context.Context) {
	if request.Method != http.MethodPost {
		writeOKFMethodNotAllowed(writer, request, http.MethodPost)
		return
	}
	data, err := readOKFBody(request.Body)
	if err != nil {
		writeOKFError(writer, err, request)
		return
	}
	value, err := decodeProfile(data)
	if err != nil {
		writeOKFError(writer, err, request)
		return
	}
	result, err := s.okf.PreviewProfile(ctx, value)
	if err != nil {
		writeOKFError(writer, err, request)
		return
	}
	writeOKFSuccess(writer, http.StatusOK, result, result.Diagnostics, "", result.Profile.Revision, request)
}
