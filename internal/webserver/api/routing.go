package api

import (
	"errors"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/storage"
	"github.com/forceu/gokapi/internal/storage/chunking"
	"net/http"
	"strconv"
	"strings"
)

type apiRoute struct {
	Url           string
	HasWildcard   bool
	ApiPerm       models.ApiPermission
	RequestParser requestParser
	execution     apiFunc
}

func (r apiRoute) Continue(w http.ResponseWriter, request requestParser, user models.User) {
	r.execution(w, request, user)
}

type apiFunc func(w http.ResponseWriter, request requestParser, user models.User)

var routes = []apiRoute{
	{
		Url:           "/files/list",
		ApiPerm:       models.ApiPermView,
		execution:     apiList,
		RequestParser: nil,
	},
	{
		Url:           "/files/list/",
		ApiPerm:       models.ApiPermView,
		execution:     apiListSingle,
		HasWildcard:   true,
		RequestParser: &paramFilesListSingle{},
	},
	{
		Url:           "/chunk/add",
		ApiPerm:       models.ApiPermUpload,
		execution:     apiChunkAdd,
		RequestParser: &paramChunkAdd{},
	},
	{
		Url:           "/chunk/complete",
		ApiPerm:       models.ApiPermUpload,
		execution:     apiChunkComplete,
		RequestParser: &paramChunkComplete{},
	},
	{
		Url:           "/files/add",
		ApiPerm:       models.ApiPermUpload,
		execution:     apiUploadFile,
		RequestParser: &paramFilesAdd{},
	},
	{
		Url:           "/files/delete",
		ApiPerm:       models.ApiPermDelete,
		execution:     apiDeleteFile,
		RequestParser: &paramFilesDelete{},
	},
	{
		Url:           "/files/duplicate",
		ApiPerm:       models.ApiPermUpload,
		execution:     apiDuplicateFile,
		RequestParser: &paramFilesDuplicate{},
	},
	{
		Url:           "/files/modify",
		ApiPerm:       models.ApiPermEdit,
		execution:     apiEditFile,
		RequestParser: &paramFilesModify{},
	},
	{
		Url:           "/files/replace",
		ApiPerm:       models.ApiPermReplace,
		execution:     apiReplaceFile,
		RequestParser: &paramFilesReplace{},
	},
	{
		Url:           "/auth/create",
		ApiPerm:       models.ApiPermApiMod,
		execution:     apiCreateApiKey,
		RequestParser: &paramAuthCreate{},
	},
	{
		Url:           "/auth/friendlyname",
		ApiPerm:       models.ApiPermApiMod,
		execution:     apiChangeFriendlyName,
		RequestParser: &paramAuthFriendlyName{},
	},
	{
		Url:           "/auth/modify",
		ApiPerm:       models.ApiPermApiMod,
		execution:     apiModifyApiKey,
		RequestParser: &paramAuthModify{},
	},
	{
		Url:           "/auth/delete",
		ApiPerm:       models.ApiPermApiMod,
		execution:     apiDeleteKey,
		RequestParser: &paramAuthDelete{},
	},
	{
		Url:           "/user/create",
		ApiPerm:       models.ApiPermManageUsers,
		execution:     apiCreateUser,
		RequestParser: &paramUserCreate{},
	},
	{
		Url:           "/user/changeRank",
		ApiPerm:       models.ApiPermManageUsers,
		execution:     apiChangeUserRank,
		RequestParser: &paramUserChangeRank{},
	},
	{
		Url:           "/user/delete",
		ApiPerm:       models.ApiPermManageUsers,
		execution:     apiDeleteUser,
		RequestParser: &paramUserDelete{},
	},
	{
		Url:           "/user/modify",
		ApiPerm:       models.ApiPermManageUsers,
		execution:     apiModifyUser,
		RequestParser: &paramUserModify{},
	},
	{
		Url:           "/user/resetPassword",
		ApiPerm:       models.ApiPermManageUsers,
		execution:     apiResetPassword,
		RequestParser: &paramUserResetPw{},
	},
	{
		Url:           "/logs/delete",
		ApiPerm:       models.ApiPermManageLogs,
		execution:     apiLogsDelete,
		RequestParser: &paramLogsDelete{},
	},
}

func getRouting(requestUrl string) (apiRoute, bool) {
	for _, route := range routes {
		if (!route.HasWildcard && requestUrl == route.Url) ||
			(route.HasWildcard && strings.HasPrefix(requestUrl, route.Url)) {
			return route, true
		}
	}
	return apiRoute{}, false
}

type requestParser interface {
	// ParseRequest reads the supplied headers, stores them and afterwards calls ProcessParameter()
	ParseRequest(r *http.Request) error
	// ProcessParameter goes through the submitted parameters, checks them and converts them to expected values
	ProcessParameter(r *http.Request) error
	// New returns an empty struct of the type
	New() requestParser
}

type paramFilesListSingle struct {
	RequestUrl string
}

func (p *paramFilesListSingle) ProcessParameter(r *http.Request) error {
	p.RequestUrl = parseRequestUrl(r)
	return nil
}

type paramFilesAdd struct {
	Request *http.Request
}

func (p *paramFilesAdd) ProcessParameter(r *http.Request) error {
	p.Request = r
	return nil
}

type paramFilesDuplicate struct {
	Id                 string `header:"id" required:"true"`
	AllowedDownloads   int    `header:"allowedDownloads"`
	ExpiryDays         int    `header:"expiryDays"`
	Password           string `header:"password"`
	KeepPassword       bool   `header:"originalPassword"`
	FileName           string `header:"filename"`
	UnlimitedDownloads bool
	UnlimitedTime      bool
	RequestedChanges   int
	foundHeaders       map[string]bool
}

func (p *paramFilesDuplicate) ProcessParameter(r *http.Request) error {
	if p.foundHeaders["allowedDownloads"] {
		p.RequestedChanges |= storage.ParamDownloads
		if p.AllowedDownloads == 0 {
			p.UnlimitedDownloads = true
		}
	}
	if p.foundHeaders["expiryDays"] {
		p.RequestedChanges |= storage.ParamExpiry
		if p.ExpiryDays == 0 {
			p.UnlimitedTime = true
		}
	}
	if !p.KeepPassword {
		if p.foundHeaders["password"] {
			p.RequestedChanges |= storage.ParamPassword
		}
	}
	if p.foundHeaders["filename"] {
		p.RequestedChanges |= storage.ParamName
	}
	return nil
}

type paramFilesModify struct {
	Id                 string `header:"id" required:"true"`
	AllowedDownloads   int    `header:"allowedDownloads"`
	ExpiryTimestamp    int64  `header:"expiryTimestamp"`
	Password           string `header:"password"`
	KeepPassword       bool   `header:"originalPassword"`
	UnlimitedDownloads bool
	UnlimitedExpiry    bool
	IsPasswordSet      bool
	foundHeaders       map[string]bool
}

func (p *paramFilesModify) ProcessParameter(_ *http.Request) error {
	if p.foundHeaders["allowedDownloads"] && p.AllowedDownloads == 0 {
		p.UnlimitedDownloads = true
	}
	if p.foundHeaders["expiryTimestamp"] && p.ExpiryTimestamp == 0 {
		p.UnlimitedExpiry = true
	}
	p.IsPasswordSet = p.foundHeaders["password"]
	return nil
}

type paramFilesReplace struct {
	Id           string `header:"id" required:"true"`
	IdNewContent string `header:"idNewContent" required:"true"`
	Delete       bool   `header:"deleteNewFile"`
	foundHeaders map[string]bool
}

func (p *paramFilesReplace) ProcessParameter(_ *http.Request) error { return nil }

type paramFilesDelete struct {
	Id           string `header:"id" required:"true"`
	foundHeaders map[string]bool
}

func (p *paramFilesDelete) ProcessParameter(_ *http.Request) error { return nil }

type paramAuthCreate struct {
	FriendlyName     string `header:"friendlyName"`
	BasicPermissions bool   `header:"basicPermissions"`
	foundHeaders     map[string]bool
}

func (p *paramAuthCreate) ProcessParameter(_ *http.Request) error { return nil }

type paramAuthFriendlyName struct {
	KeyId        string `header:"targetKey" required:"true"`
	FriendlyName string `header:"friendlyName" required:"true"`
	foundHeaders map[string]bool
}

func (p *paramAuthFriendlyName) ProcessParameter(_ *http.Request) error { return nil }

type paramAuthModify struct {
	KeyId              string `header:"targetKey" required:"true"`
	permissionRaw      string `header:"permission" required:"true"`
	permissionModifier string `header:"permissionModifier" required:"true"`
	Permission         models.ApiPermission
	GrantPermission    bool
	foundHeaders       map[string]bool
}

func (p *paramAuthModify) ProcessParameter(_ *http.Request) error {
	switch strings.ToUpper(p.permissionRaw) {
	case "PERM_VIEW":
		p.Permission = models.ApiPermView
	case "PERM_UPLOAD":
		p.Permission = models.ApiPermUpload
	case "PERM_DELETE":
		p.Permission = models.ApiPermDelete
	case "PERM_API_MOD":
		p.Permission = models.ApiPermApiMod
	case "PERM_EDIT":
		p.Permission = models.ApiPermEdit
	case "PERM_REPLACE":
		p.Permission = models.ApiPermReplace
	case "PERM_MANAGE_USERS":
		p.Permission = models.ApiPermManageUsers
	case "PERM_MANAGE_LOGS":
		p.Permission = models.ApiPermManageLogs
	default:
		return errors.New("invalid permission")
	}
	switch strings.ToUpper(p.permissionModifier) {
	case "GRANT":
		p.GrantPermission = true
	case "REVOKE":
		p.GrantPermission = false
	default:
		return errors.New("invalid permission modifier")
	}
	return nil
}

type paramAuthDelete struct {
	KeyId        string `header:"targetKey" required:"true"`
	foundHeaders map[string]bool
}

func (p *paramAuthDelete) ProcessParameter(_ *http.Request) error { return nil }

type paramUserCreate struct {
	Username     string `header:"username" required:"true"`
	foundHeaders map[string]bool
}

func (p *paramUserCreate) ProcessParameter(_ *http.Request) error { return nil }

type paramUserChangeRank struct {
	Id           int    `header:"userid" required:"true"`
	newRankRaw   string `header:"newRank" required:"true"`
	NewRank      models.UserRank
	foundHeaders map[string]bool
}

func (p *paramUserChangeRank) ProcessParameter(_ *http.Request) error {
	switch strings.ToLower(p.newRankRaw) {
	case "admin":
		p.NewRank = models.UserLevelAdmin
	case "user":
		p.NewRank = models.UserLevelUser
	default:
		return errors.New("invalid rank")
	}
	return nil
}

type paramUserDelete struct {
	Id           int  `header:"userid" required:"true"`
	DeleteFiles  bool `header:"deleteFiles"`
	foundHeaders map[string]bool
}

func (p *paramUserDelete) ProcessParameter(_ *http.Request) error { return nil }

type paramUserModify struct {
	Id                 int `header:"userid" required:"true"`
	Permission         models.UserPermission
	permissionRaw      string `header:"userpermission" required:"true"`
	permissionModifier string `header:"permissionModifier" required:"true"`
	GrantPermission    bool
	foundHeaders       map[string]bool
}

func (p *paramUserModify) ProcessParameter(_ *http.Request) error {
	switch strings.ToUpper(p.permissionRaw) {
	case "PERM_REPLACE":
		p.Permission = models.UserPermReplaceUploads
	case "PERM_LIST":
		p.Permission = models.UserPermListOtherUploads
	case "PERM_EDIT":
		p.Permission = models.UserPermEditOtherUploads
	case "PERM_REPLACE_OTHER":
		p.Permission = models.UserPermReplaceOtherUploads
	case "PERM_DELETE":
		p.Permission = models.UserPermDeleteOtherUploads
	case "PERM_LOGS":
		p.Permission = models.UserPermManageLogs
	case "PERM_API":
		p.Permission = models.UserPermManageApiKeys
	case "PERM_USERS":
		p.Permission = models.UserPermManageUsers
	default:
		return errors.New("invalid permission")
	}
	switch strings.ToUpper(p.permissionModifier) {
	case "GRANT":
		p.GrantPermission = true
	case "REVOKE":
		p.GrantPermission = false
	default:
		return errors.New("invalid permission modifier")
	}
	return nil
}

type paramUserResetPw struct {
	Id           int  `header:"userid"  required:"true"`
	NewPassword  bool `header:"generateNewPassword"`
	foundHeaders map[string]bool
}

func (p *paramUserResetPw) ProcessParameter(_ *http.Request) error { return nil }

type paramLogsDelete struct {
	Timestamp    int64 `header:"timestamp"`
	Request      *http.Request
	foundHeaders map[string]bool
}

func (p *paramLogsDelete) ProcessParameter(r *http.Request) error {
	p.Request = r
	return nil
}

type paramChunkAdd struct {
	Request *http.Request
}

func (p *paramChunkAdd) ProcessParameter(r *http.Request) error {
	p.Request = r
	return nil
}

type paramChunkComplete struct {
	Uuid               string `header:"uuid" required:"true"`
	FileName           string `header:"filename" required:"true"`
	FileSize           int64  `header:"filesize" required:"true"`
	RealSize           int64  `header:"realsize"`
	ContentType        string `header:"contenttype"`
	AllowedDownloads   int    `header:"allowedDownloads"`
	ExpiryDays         int    `header:"expiryDays"`
	Password           string `header:"password"`
	IsE2E              bool   `header:"isE2E"`
	IsNonBlocking      bool   `header:"nonblocking"`
	UnlimitedDownloads bool
	UnlimitedTime      bool
	FileHeader         chunking.FileHeader
	foundHeaders       map[string]bool
}

func (p *paramChunkComplete) ProcessParameter(_ *http.Request) error {

	if !p.foundHeaders["realsize"] {
		if !p.IsE2E {
			p.RealSize = p.FileSize
		} else {
			return errors.New("e2e set, but realsize not submitted")
		}
	}

	if p.foundHeaders["allowedDownloads"] && p.AllowedDownloads == 0 {
		p.UnlimitedDownloads = true
	}
	if p.foundHeaders["expiryDays"] && p.ExpiryDays == 0 {
		p.UnlimitedTime = true
	}
	p.FileHeader = chunking.FileHeader{
		Filename:    p.FileName,
		ContentType: p.ContentType,
		Size:        p.FileSize,
	}
	return nil
}

func checkHeaderExists(r *http.Request, key string, isRequired, isString bool) (bool, error) {
	if r.Header.Get(key) != "" {
		return true, nil
	}
	if isRequired {
		return false, errors.New("header " + key + " is required")
	}
	if isString {
		return len(r.Header.Values(key)) > 0, nil
	}
	return false, nil
}

func parseHeaderBool(r *http.Request, key string) (bool, error) {
	value, err := strconv.ParseBool(r.Header.Get(key))
	if err != nil {
		return false, err
	}
	return value, nil
}

func parseHeaderInt64(r *http.Request, key string) (int64, error) {
	value, err := strconv.ParseInt(r.Header.Get(key), 10, 64)
	if err != nil {
		return 0, err
	}
	return value, nil
}

func parseHeaderInt(r *http.Request, key string) (int, error) {
	value, err := strconv.Atoi(r.Header.Get(key))
	if err != nil {
		return 0, err
	}
	return value, nil
}
