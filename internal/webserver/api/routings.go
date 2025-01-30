package api

import (
	"errors"
	"github.com/forceu/gokapi/internal/models"
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

type apiRoute struct {
	Url         string
	HasWildcard bool
	ApiPerm     models.ApiPermission
	Parsing     paramInfo
	execution   apiFunc
}

func (r apiRoute) Continue(w http.ResponseWriter, request paramInfo, user models.User) {
	r.execution(w, request, user)
}

type apiFunc func(w http.ResponseWriter, request paramInfo, user models.User)

var routes = []apiRoute{
	{
		Url:       "/files/list",
		ApiPerm:   models.ApiPermView,
		execution: apiList,
		Parsing:   nil,
	},
	{
		Url:         "/files/list/",
		ApiPerm:     models.ApiPermView,
		execution:   apiListSingle,
		HasWildcard: true,
		Parsing:     &paramFilesListSingle{},
	},
	{
		Url:       "/chunk/add",
		ApiPerm:   models.ApiPermUpload,
		execution: apiChunkAdd,
		Parsing:   &paramChunkAdd{},
	},
	{
		Url:       "/chunk/complete",
		ApiPerm:   models.ApiPermUpload,
		execution: apiChunkComplete,
		Parsing:   &paramChunkComplete{},
	},
	{
		Url:       "/files/add",
		ApiPerm:   models.ApiPermUpload,
		execution: apiUploadFile,
		Parsing:   &paramFilesAdd{},
	},
	{
		Url:       "/files/delete",
		ApiPerm:   models.ApiPermDelete,
		execution: apiDeleteFile,
		Parsing:   &paramFilesDelete{},
	},
	{
		Url:       "/files/duplicate",
		ApiPerm:   models.ApiPermUpload,
		execution: apiDuplicateFile,
		Parsing:   &paramFilesDuplicate{},
	},
	{
		Url:       "/files/modify",
		ApiPerm:   models.ApiPermEdit,
		execution: apiEditFile,
		Parsing:   &paramFilesModify{},
	},
	{
		Url:       "/files/replace",
		ApiPerm:   models.ApiPermReplace,
		execution: apiReplaceFile,
		Parsing:   &paramFilesReplace{},
	},
	{
		Url:       "/auth/create",
		ApiPerm:   models.ApiPermApiMod,
		execution: apiCreateApiKey,
		Parsing:   &paramAuthCreate{},
	},
	{
		Url:       "/auth/friendlyname",
		ApiPerm:   models.ApiPermApiMod,
		execution: apiChangeFriendlyName,
		Parsing:   &paramAuthFriendlyName{},
	},
	{
		Url:       "/auth/modify",
		ApiPerm:   models.ApiPermApiMod,
		execution: apiModifyApiKey,
		Parsing:   &paramAuthModify{},
	},
	{
		Url:       "/auth/delete",
		ApiPerm:   models.ApiPermApiMod,
		execution: apiDeleteKey,
		Parsing:   &paramAuthDelete{},
	},
	{
		Url:       "/user/create",
		ApiPerm:   models.ApiPermManageUsers,
		execution: apiCreateUser,
		Parsing:   &paramUserCreate{},
	},
	{
		Url:       "/user/changeRank",
		ApiPerm:   models.ApiPermManageUsers,
		execution: apiChangeUserRank,
		Parsing:   &paramUserChangeRank{},
	},
	{
		Url:       "/user/delete",
		ApiPerm:   models.ApiPermManageUsers,
		execution: apiDeleteUser,
		Parsing:   &paramUserDelete{},
	},
	{
		Url:       "/user/modify",
		ApiPerm:   models.ApiPermManageUsers,
		execution: apiModifyUser,
		Parsing:   &paramUserModify{},
	},
	{
		Url:       "/user/resetPassword",
		ApiPerm:   models.ApiPermManageUsers,
		execution: apiResetPassword,
		Parsing:   &paramUserResetPw{},
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

type paramInfo interface {
	ProcessParameter([]string) error
}

type paramFilesListSingle struct {
	RequestUrl string `isRequestUrl:"true"`
}

func (p *paramFilesListSingle) ProcessParameter(_ []string) error {
	return nil
}

type paramFilesAdd struct {
	File               string `header:"file" required:"true"`
	AllowedDownloads   int    `header:"allowedDownloads"`
	ExpiryDays         int    `header:"expiryDays"`
	Password           string `header:"password"`
	UnlimitedDownloads bool
	UnlimitedExpiry    bool
}

func (p *paramFilesDuplicate) paramFilesAdd(fields []string) error {
	if slices.Contains(fields, "allowedDownloads") && p.AllowedDownloads == 0 {
		p.UnlimitedDownloads = true
	}
	if slices.Contains(fields, "expiryDays") && p.ExpiryDays == 0 {
		p.UnlimitedExpiry = true
	}
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
	UnlimitedExpiry    bool
	IsPasswordSet      bool
}

func (p *paramFilesDuplicate) ProcessParameter(fields []string) error {
	if slices.Contains(fields, "allowedDownloads") && p.AllowedDownloads == 0 {
		p.UnlimitedDownloads = true
	}
	if slices.Contains(fields, "expiryDays") && p.ExpiryDays == 0 {
		p.UnlimitedExpiry = true
	}
	p.IsPasswordSet = slices.Contains(fields, "password")
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
}

func (p *paramFilesModify) ProcessParameter(fields []string) error {
	if slices.Contains(fields, "allowedDownloads") && p.AllowedDownloads == 0 {
		p.UnlimitedDownloads = true
	}
	if slices.Contains(fields, "expiryDays") && p.ExpiryTimestamp == 0 {
		p.UnlimitedExpiry = true
	}
	p.IsPasswordSet = slices.Contains(fields, "password")
	return nil
}

type paramFilesReplace struct {
	Id           string `header:"id" required:"true"`
	IdNewContent string `header:"idNewContent" required:"true"`
	Delete       bool   `header:"deleteNewFile"`
}

func (p *paramFilesReplace) ProcessParameter(_ []string) error { return nil }

type paramFilesDelete struct {
	Id string `header:"id" required:"true"`
}

func (p *paramFilesDelete) ProcessParameter(_ []string) error { return nil }

type paramAuthCreate struct {
	FriendlyName     string `header:"friendlyName"`
	BasicPermissions bool   `header:"basicPermissions"`
}

func (p *paramAuthCreate) ProcessParameter(_ []string) error { return nil }

type paramAuthFriendlyName struct {
	KeyId        string `header:"apiKeyToModify" required:"true"`
	FriendlyName string `header:"friendlyName" required:"true"`
}

func (p *paramAuthFriendlyName) ProcessParameter(_ []string) error { return nil }

type paramAuthModify struct {
	KeyId           string `header:"apiKeyToModify" required:"true"`
	Permission      models.ApiPermission
	permissionRaw   string `header:"permission" required:"true"`
	GrantPermission bool   `header:"permissionModifier" required:"true"`
}

func (p *paramAuthModify) ProcessParameter(_ []string) error {
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
	default:
		return errors.New("invalid permission")
	}
	return nil
}

type paramAuthDelete struct {
	KeyId string `header:"apiKeyToModify" required:"true"`
}

func (p *paramAuthDelete) ProcessParameter(_ []string) error { return nil }

type paramUserCreate struct {
	Username string `header:"username" required:"true"`
}

func (p *paramUserCreate) ProcessParameter(_ []string) error { return nil }

type paramUserChangeRank struct {
	Id         int `header:"userid" required:"true"`
	NewRank    models.UserRank
	newRankRaw string `header:"newRank" required:"true"`
}

func (p *paramUserChangeRank) ProcessParameter(_ []string) error {
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
	Id          int  `header:"userid" required:"true"`
	DeleteFiles bool `header:"deleteFiles"`
}

func (p *paramUserDelete) ProcessParameter(_ []string) error { return nil }

type paramUserModify struct {
	Id              int `header:"userid" required:"true"`
	Permission      models.UserPermission
	permissionRaw   string `header:"userpermission" required:"true"`
	GrantPermission bool   `header:"permissionModifier" required:"true"`
}

func (p *paramUserModify) ProcessParameter(_ []string) error {
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
	return nil
}

type paramUserResetPw struct {
	Id          int  `header:"userid"  required:"true"`
	NewPassword bool `header:"generateNewPassword"`
}

func (p *paramUserResetPw) ProcessParameter(_ []string) error { return nil }

type paramChunkAdd struct {
	File     string `header:"file" required:"true"`
	Uuid     string `header:"uuid" required:"true"`
	FileSize int    `header:"filesize" required:"true"`
	Offset   int    `header:"offset" required:"true"`
}

func (p *paramChunkAdd) ProcessParameter(_ []string) error { return nil }

type paramChunkComplete struct {
	Uuid               string `header:"uuid" required:"true"`
	FileName           string `header:"filename" required:"true"`
	FileSize           int    `header:"filesize" required:"true"`
	ContentType        string `header:"contenttype"`
	AllowedDownloads   int    `header:"allowedDownloads"`
	ExpiryDays         int    `header:"expiryDays"`
	Password           string `header:"password"`
	UnlimitedDownloads bool
	UnlimitedExpiry    bool
}

func (p *paramChunkComplete) ProcessParameter(fields []string) error {
	if slices.Contains(fields, "allowedDownloads") && p.AllowedDownloads == 0 {
		p.UnlimitedDownloads = true
	}
	if slices.Contains(fields, "expiryDays") && p.ExpiryDays == 0 {
		p.UnlimitedExpiry = true
	}
	return nil
}

func parseParameters(input *paramInfo, r *http.Request) error {
	fields := reflect.TypeOf(input)
	values := reflect.ValueOf(input)
	amountFields := fields.NumField()

	processedFields := make([]string, 0)

	for i := 0; i < amountFields; i++ {
		field := fields.Field(i)
		value := values.Field(i)

		_, isRequestUrl := field.Tag.Lookup("`isRequestUrl")
		if isRequestUrl {
			requestUrl := strings.Replace(r.URL.String(), "/api", "", 1)
			values.Field(i).SetString(requestUrl)
			continue
		}

		headerName, isLookupTag := field.Tag.Lookup("headerName")
		if !isLookupTag {
			continue
		}
		_, isRequired := field.Tag.Lookup("required")
		headerValue := r.Header.Get(headerName)
		if headerValue == "" {
			if isRequired {
				return errors.New("parameter " + headerName + " not set")
			} else {
				continue
			}
		}
		processedFields = append(processedFields, headerName)
		switch value.Kind() {
		case reflect.String:
			values.Field(i).SetString(headerValue)
		case reflect.Int:
			v, err := strconv.Atoi(headerValue)
			if err != nil {
				return err
			}
			values.Field(i).SetInt(int64(v))
		case reflect.Int64:
			v, err := strconv.ParseInt(headerValue, 10, 64)
			if err != nil {
				return err
			}
			values.Field(i).SetInt(v)
		case reflect.Bool:
			b, err := strconv.ParseBool(headerValue)
			if err != nil {
				return err
			}
			values.Field(i).SetBool(b)
		default:
			panic("struct is not supported")
		}
	}
	return (*input).ProcessParameter(processedFields)
}
