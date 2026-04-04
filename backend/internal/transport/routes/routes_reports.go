package routes

import (
	"net/http"

	"github.com/tlmanz/allure-hub/internal/transport/handler"
	localauth "github.com/tlmanz/allure-hub/pkg/authkit"
	kit "github.com/tlmanz/authkit"
)

func RegisterReportRoutes(mux *http.ServeMux, auth *kit.Auth, rh *handler.ReportHandler, uh *handler.UploadSessionHandler) {
	// Results upload — API keys allowed (primary CI/CD path)
	mux.Handle("POST /api/environments/{envId}/projects/{projectId}/results", auth.Require(localauth.PermUpload)(http.HandlerFunc(rh.UploadResultsStream)))
	mux.Handle("POST /api/environments/{envId}/projects/{projectId}/uploads", auth.Require(localauth.PermUpload)(http.HandlerFunc(rh.InitChunkedUpload)))
	mux.Handle("PUT /api/environments/{envId}/projects/{projectId}/uploads/{uploadId}", auth.Require(localauth.PermUpload)(http.HandlerFunc(rh.UploadChunk)))
	mux.Handle("POST /api/environments/{envId}/projects/{projectId}/uploads/{uploadId}/complete", auth.Require(localauth.PermUpload)(http.HandlerFunc(rh.CompleteChunkedUpload)))

	// Report generation + listing
	mux.Handle("POST /api/environments/{envId}/projects/{projectId}/reports", auth.Require(localauth.PermUpload)(http.HandlerFunc(rh.GenerateReport)))
	mux.Handle("GET /api/environments/{envId}/projects/{projectId}/reports", auth.Require(localauth.PermView)(http.HandlerFunc(rh.ListReports)))
	mux.Handle("GET /api/environments/{envId}/projects/{projectId}/reports/stats", auth.Require(localauth.PermView)(http.HandlerFunc(rh.ReportStats)))
	mux.Handle("DELETE /api/environments/{envId}/projects/{projectId}/reports/{buildId}", auth.RequireSession(localauth.PermManage)(http.HandlerFunc(rh.DeleteReport)))

	// Upload session tracking
	mux.Handle("GET /api/uploads", auth.Require(localauth.PermView)(http.HandlerFunc(uh.List)))
	mux.Handle("GET /api/uploads/stream", auth.Require(localauth.PermView)(http.HandlerFunc(uh.Stream)))
	mux.Handle("DELETE /api/uploads/{id}", auth.RequireSession(localauth.PermManage)(http.HandlerFunc(uh.Delete)))
}
