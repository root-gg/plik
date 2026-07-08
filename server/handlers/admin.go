package handlers

import (
	"net/http"
	"strconv"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	"github.com/root-gg/plik/server/metadata"
)

// GetUsers return users
func GetUsers(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {

	// Double check authorization
	if !ctx.IsAdmin() {
		ctx.Forbidden("you need administrator privileges")
		return
	}

	pagingQuery := ctx.GetPagingQuery()
	sort := req.URL.Query().Get("sort")
	if !isUserSort(sort) {
		ctx.BadRequest("invalid sort")
		return
	}

	provider := req.URL.Query().Get("provider")

	var admin *bool
	if adminStr := req.URL.Query().Get("admin"); adminStr != "" {
		isAdmin := adminStr == "true"
		admin = &isAdmin
	}

	// Get users
	users, cursor, err := ctx.GetMetadataBackend().GetUsers(provider, admin, false, sort, pagingQuery)
	if err != nil {
		ctx.InternalServerError("unable to get users : %s", err)
		return
	}

	// Count total users matching the filters
	// Note: not in the same transaction as the paginated query above, so the total
	// may be slightly inconsistent with the results if users are created/deleted
	// concurrently. This is acceptable for an admin UI counter.
	total, err := ctx.GetMetadataBackend().CountUsers(provider, admin)
	if err != nil {
		ctx.InternalServerError("unable to count users : %s", err)
		return
	}

	pagingResponse := common.NewPagingResponse(users, cursor).WithTotal(total)
	common.WriteJSONResponse(resp, pagingResponse)
}

// isUsageStatsSort validates the public user/token stats "sort" query
// parameter. The empty value is accepted by list handlers as default date
// ordering; size and lifetimeSize select counter-backed current/lifetime usage.
// Shared by GET /users and GET /me/token — see isUserSort for the wider list
// GET /users alone accepts.
func isUsageStatsSort(sort string) bool {
	switch sort {
	case "", metadata.StatsSortDate, metadata.StatsSortSize, metadata.StatsSortLifetimeSize:
		return true
	default:
		return false
	}
}

// isUserSort validates the "sort" query parameter for GET /users. It extends
// isUsageStatsSort with downloadedBytes (per-user lifetime downloaded bytes,
// usage_stats.downloaded_bytes via the getUsersSortedByUsage join). This is
// intentionally NOT folded into isUsageStatsSort itself: that function is
// shared with GET /me/token (server/handlers/me.go), whose GetTokens/
// getTokensSortedByUsage switch does not implement downloadedBytes — sharing
// the wider list there would turn a clean 400 into a 500 for
// /me/token?sort=downloadedBytes.
func isUserSort(sort string) bool {
	if sort == metadata.StatsSortDownloadedBytes {
		return true
	}
	return isUsageStatsSort(sort)
}

// SearchUsers search users by query string
func SearchUsers(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {

	// Double check authorization
	if !ctx.IsAdmin() {
		ctx.Forbidden("you need administrator privileges")
		return
	}

	q := req.URL.Query().Get("q")
	if len(q) < 2 {
		ctx.BadRequest("search query must be at least 2 characters")
		return
	}

	provider := req.URL.Query().Get("provider")

	var admin *bool
	if adminStr := req.URL.Query().Get("admin"); adminStr != "" {
		isAdmin := adminStr == "true"
		admin = &isAdmin
	}

	limit := 5
	if limitStr := req.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	if limit > 20 {
		limit = 20
	}

	users, err := ctx.GetMetadataBackend().SearchUsers(q, provider, admin, limit)
	if err != nil {
		ctx.InternalServerError("unable to search users : %s", err)
		return
	}

	common.WriteJSONResponse(resp, users)
}

// GetUploads return uploads
func GetUploads(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	// Double check authorization
	if !ctx.IsAdmin() {
		ctx.Forbidden("you need administrator privileges")
		return
	}

	pagingQuery := ctx.GetPagingQuery()
	sort := req.URL.Query().Get("sort")
	if !isUploadSort(sort) {
		ctx.BadRequest("invalid sort")
		return
	}

	filters := parseBadgeFilters(req)
	filters.User = req.URL.Query().Get("user")
	filters.Token = req.URL.Query().Get("token")

	uploads, cursor, err := getUploadsSorted(ctx, sort, filters, pagingQuery)
	if err != nil {
		ctx.InternalServerError("unable to get uploads : %s", err)
		return
	}

	// Count total uploads matching the filters
	// Note: not in the same transaction as the paginated query above, so the total
	// may be slightly inconsistent with the results if uploads are cleaned up
	// concurrently. This is acceptable for an admin UI counter.
	total, err := ctx.GetMetadataBackend().CountUploads(filters)
	if err != nil {
		ctx.InternalServerError("unable to count uploads : %s", err)
		return
	}

	pagingResponse := common.NewPagingResponse(uploads, cursor).WithTotal(total)
	common.WriteJSONResponse(resp, pagingResponse)
}

// GetServerStatistics return the server statistics
func GetServerStatistics(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {

	// Double check authorization
	if !ctx.IsAdmin() {
		ctx.Forbidden("you need administrator privileges")
		return
	}

	// Get server statistics
	stats, err := ctx.GetMetadataBackend().GetServerStatistics()
	if err != nil {
		ctx.InternalServerError("unable to get server statistics : %s", err)
		return
	}

	common.WriteJSONResponse(resp, stats)
}

// GetServerActivityDaily returns the admin-only server-wide daily activity
// series (downloads, downloaded bytes, uploads, uploaded bytes) for the last N
// UTC days (oldest first, dense, zero-filled).
func GetServerActivityDaily(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	if !ctx.IsAdmin() {
		ctx.Forbidden("you need administrator privileges")
		return
	}

	days, err := parseActivityDailyDays(req)
	if err != nil {
		ctx.BadRequest("%s", err)
		return
	}

	points, err := ctx.GetMetadataBackend().GetServerActivityStatsDaily(days)
	if err != nil {
		ctx.InternalServerError("unable to get server activity stats", err)
		return
	}

	if points == nil {
		points = []*common.ActivityDailyPoint{}
	}

	common.WriteJSONResponse(resp, points)
}

// parseActivityDailyDays validates the "days" query parameter shared by the two
// activity-series endpoints. It defaults to 30, requires an integer, and enforces
// the 1..31 range, returning a curated 400 otherwise. This handler is the single
// validation site; the metadata series methods trust their inputs.
func parseActivityDailyDays(req *http.Request) (int, error) {
	days := 30
	if daysStr := req.URL.Query().Get("days"); daysStr != "" {
		parsed, err := strconv.Atoi(daysStr)
		if err != nil {
			return 0, common.NewHTTPError("invalid days: must be an integer between 1 and 31", nil, http.StatusBadRequest)
		}
		days = parsed
	}
	if days < 1 || days > 31 {
		return 0, common.NewHTTPError("invalid days: must be between 1 and 31", nil, http.StatusBadRequest)
	}
	return days, nil
}

// GetTrendingUploads returns admin-only upload trending statistics. Shares its
// window/limit/sort parsing with GET /me/stats/trending/uploads (me.go) via
// the misc.go helpers.
func GetTrendingUploads(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	if !ctx.IsAdmin() {
		ctx.Forbidden("you need administrator privileges")
		return
	}

	limit, err := parseTrendingLimit(req)
	if err != nil {
		ctx.BadRequest("%s", err)
		return
	}
	window, err := parseTrendingWindow(req)
	if err != nil {
		ctx.BadRequest("%s", err)
		return
	}
	sort, err := parseTrendingSort(req)
	if err != nil {
		ctx.BadRequest("%s", err)
		return
	}

	items, err := ctx.GetMetadataBackend().GetTrendingUploads(window, sort, limit)
	if err != nil {
		ctx.InternalServerError("unable to get trending uploads", err)
		return
	}

	if items == nil {
		items = []*common.TrendingItem{}
	}

	common.WriteJSONResponse(resp, items)
}

// GetTrendingFiles returns admin-only file trending statistics. There is no
// sort param here — file-grain byte trending is out of scope — so
// Trending Files only ever ranks by download count.
func GetTrendingFiles(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	if !ctx.IsAdmin() {
		ctx.Forbidden("you need administrator privileges")
		return
	}

	limit, err := parseTrendingLimit(req)
	if err != nil {
		ctx.BadRequest("%s", err)
		return
	}
	window, err := parseTrendingWindow(req)
	if err != nil {
		ctx.BadRequest("%s", err)
		return
	}

	items, err := ctx.GetMetadataBackend().GetTrendingFiles(window, limit)
	if err != nil {
		ctx.InternalServerError("unable to get trending files", err)
		return
	}

	if items == nil {
		items = []*common.TrendingItem{}
	}

	common.WriteJSONResponse(resp, items)
}
