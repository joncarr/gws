package google

import (
	"context"
	"fmt"
	"strings"

	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/option"

	"github.com/joncarr/gws/internal/auth"
	"github.com/joncarr/gws/internal/config"
)

type DomainInfo struct {
	CustomerID           string `json:"customer_id"`
	PrimaryDomain        string `json:"primary_domain"`
	AlternateEmail       string `json:"alternate_email,omitempty"`
	CustomerCreationTime string `json:"customer_creation_time,omitempty"`
	Language             string `json:"language,omitempty"`
	PhoneNumber          string `json:"phone_number,omitempty"`
	PostalAddress        any    `json:"postal_address,omitempty"`
	Etag                 string `json:"etag,omitempty"`
	Kind                 string `json:"kind,omitempty"`
}

type WorkspaceDomainInfo struct {
	DomainName    string            `json:"domain_name"`
	IsPrimary     bool              `json:"is_primary"`
	Verified      bool              `json:"verified"`
	CreationTime  int64             `json:"creation_time,omitempty"`
	DomainAliases []DomainAliasInfo `json:"domain_aliases,omitempty"`
	Etag          string            `json:"etag,omitempty"`
	Kind          string            `json:"kind,omitempty"`
}

type DomainAliasInfo struct {
	DomainAliasName  string `json:"domain_alias_name"`
	ParentDomainName string `json:"parent_domain_name,omitempty"`
	Verified         bool   `json:"verified"`
	CreationTime     int64  `json:"creation_time,omitempty"`
	Etag             string `json:"etag,omitempty"`
	Kind             string `json:"kind,omitempty"`
}

type UserInfo struct {
	ID                         string         `json:"id,omitempty"`
	CustomerID                 string         `json:"customer_id,omitempty"`
	PrimaryEmail               string         `json:"primary_email"`
	Name                       string         `json:"name,omitempty"`
	GivenName                  string         `json:"given_name,omitempty"`
	FamilyName                 string         `json:"family_name,omitempty"`
	Aliases                    []string       `json:"aliases,omitempty"`
	NonEditableAliases         []string       `json:"non_editable_aliases,omitempty"`
	Suspended                  bool           `json:"suspended"`
	SuspensionReason           string         `json:"suspension_reason,omitempty"`
	IsArchived                 bool           `json:"is_archived"`
	OrgUnitPath                string         `json:"org_unit_path,omitempty"`
	IsAdmin                    bool           `json:"is_admin"`
	IsDelegatedAdmin           bool           `json:"is_delegated_admin"`
	IsEnrolledIn2SV            bool           `json:"is_enrolled_in_2sv"`
	IsEnforcedIn2SV            bool           `json:"is_enforced_in_2sv"`
	IsMailboxSetup             bool           `json:"is_mailbox_setup"`
	IsGuestUser                bool           `json:"is_guest_user"`
	IncludeInGlobalAddressList bool           `json:"include_in_global_address_list"`
	AgreedToTerms              bool           `json:"agreed_to_terms"`
	ChangePasswordAtNextLogin  bool           `json:"change_password_at_next_login"`
	IPWhitelisted              bool           `json:"ip_whitelisted"`
	CreationTime               string         `json:"creation_time,omitempty"`
	LastLoginTime              string         `json:"last_login_time,omitempty"`
	DeletionTime               string         `json:"deletion_time,omitempty"`
	RecoveryEmail              string         `json:"recovery_email,omitempty"`
	RecoveryPhone              string         `json:"recovery_phone,omitempty"`
	ThumbnailPhotoURL          string         `json:"thumbnail_photo_url,omitempty"`
	ThumbnailPhotoEtag         string         `json:"thumbnail_photo_etag,omitempty"`
	Emails                     any            `json:"emails,omitempty"`
	Phones                     any            `json:"phones,omitempty"`
	Addresses                  any            `json:"addresses,omitempty"`
	Organizations              any            `json:"organizations,omitempty"`
	Relations                  any            `json:"relations,omitempty"`
	ExternalIDs                any            `json:"external_ids,omitempty"`
	Locations                  any            `json:"locations,omitempty"`
	CustomSchemas              map[string]any `json:"custom_schemas,omitempty"`
}

type GroupInfo struct {
	Email              string   `json:"email"`
	ID                 string   `json:"id,omitempty"`
	Name               string   `json:"name,omitempty"`
	Description        string   `json:"description,omitempty"`
	DirectMembersCount int64    `json:"direct_members_count"`
	AdminCreated       bool     `json:"admin_created"`
	Aliases            []string `json:"aliases,omitempty"`
	NonEditableAliases []string `json:"non_editable_aliases,omitempty"`
	Etag               string   `json:"etag,omitempty"`
	Kind               string   `json:"kind,omitempty"`
}

type AliasInfo struct {
	Alias        string `json:"alias"`
	PrimaryEmail string `json:"primary_email,omitempty"`
	ID           string `json:"id,omitempty"`
	Etag         string `json:"etag,omitempty"`
	Kind         string `json:"kind,omitempty"`
}

type MemberInfo struct {
	Email            string `json:"email"`
	ID               string `json:"id,omitempty"`
	Role             string `json:"role,omitempty"`
	Type             string `json:"type,omitempty"`
	Status           string `json:"status,omitempty"`
	DeliverySettings string `json:"delivery_settings,omitempty"`
	Etag             string `json:"etag,omitempty"`
	Kind             string `json:"kind,omitempty"`
}

type OrgUnitInfo struct {
	Name              string `json:"name"`
	OrgUnitID         string `json:"org_unit_id,omitempty"`
	OrgUnitPath       string `json:"org_unit_path"`
	ParentOrgUnitID   string `json:"parent_org_unit_id,omitempty"`
	ParentOrgUnitPath string `json:"parent_org_unit_path,omitempty"`
	Description       string `json:"description,omitempty"`
	BlockInheritance  bool   `json:"block_inheritance"`
	Etag              string `json:"etag,omitempty"`
	Kind              string `json:"kind,omitempty"`
}

type OrgUnitCreate struct {
	Name              string
	ParentOrgUnitPath string
	Description       string
}

type OrgUnitUpdate struct {
	Name              string
	ParentOrgUnitPath string
	Description       string
}

type DirectoryClient interface {
	DomainInfo(ctx context.Context, profile config.Profile) (DomainInfo, error)
	Domains(ctx context.Context, profile config.Profile) ([]WorkspaceDomainInfo, error)
	Domain(ctx context.Context, profile config.Profile, domainName string) (WorkspaceDomainInfo, error)
	CreateDomain(ctx context.Context, profile config.Profile, domainName string) (WorkspaceDomainInfo, error)
	DeleteDomain(ctx context.Context, profile config.Profile, domainName string) error
	DomainAliases(ctx context.Context, profile config.Profile) ([]DomainAliasInfo, error)
	DomainAlias(ctx context.Context, profile config.Profile, domainAliasName string) (DomainAliasInfo, error)
	CreateDomainAlias(ctx context.Context, profile config.Profile, parentDomainName string, domainAliasName string) (DomainAliasInfo, error)
	DeleteDomainAlias(ctx context.Context, profile config.Profile, domainAliasName string) error
	Users(ctx context.Context, profile config.Profile, opts UserListOptions) ([]UserInfo, error)
	User(ctx context.Context, profile config.Profile, email string) (UserInfo, error)
	CreateUser(ctx context.Context, profile config.Profile, create UserCreate) (UserInfo, error)
	UpdateUser(ctx context.Context, profile config.Profile, email string, update UserUpdate) (UserInfo, error)
	DeleteUser(ctx context.Context, profile config.Profile, email string) error
	SetUserAdmin(ctx context.Context, profile config.Profile, email string, admin bool) error
	SetUserSuspended(ctx context.Context, profile config.Profile, email string, suspended bool) (UserInfo, error)
	UserAliases(ctx context.Context, profile config.Profile, userKey string) ([]AliasInfo, error)
	CreateUserAlias(ctx context.Context, profile config.Profile, userKey string, alias string) (AliasInfo, error)
	DeleteUserAlias(ctx context.Context, profile config.Profile, userKey string, alias string) error
	Groups(ctx context.Context, profile config.Profile, opts GroupListOptions) ([]GroupInfo, error)
	Group(ctx context.Context, profile config.Profile, email string) (GroupInfo, error)
	CreateGroup(ctx context.Context, profile config.Profile, group GroupInfo) (GroupInfo, error)
	UpdateGroup(ctx context.Context, profile config.Profile, email string, group GroupInfo) (GroupInfo, error)
	DeleteGroup(ctx context.Context, profile config.Profile, email string) error
	GroupAliases(ctx context.Context, profile config.Profile, groupKey string) ([]AliasInfo, error)
	CreateGroupAlias(ctx context.Context, profile config.Profile, groupKey string, alias string) (AliasInfo, error)
	DeleteGroupAlias(ctx context.Context, profile config.Profile, groupKey string, alias string) error
	GroupMembers(ctx context.Context, profile config.Profile, groupEmail string, limit int64) ([]MemberInfo, error)
	GroupMember(ctx context.Context, profile config.Profile, groupEmail string, memberEmail string) (MemberInfo, error)
	AddGroupMember(ctx context.Context, profile config.Profile, groupEmail string, memberEmail string, role string) (MemberInfo, error)
	UpdateGroupMember(ctx context.Context, profile config.Profile, groupEmail string, memberEmail string, role string) (MemberInfo, error)
	RemoveGroupMember(ctx context.Context, profile config.Profile, groupEmail string, memberEmail string) error
	OrgUnits(ctx context.Context, profile config.Profile) ([]OrgUnitInfo, error)
	OrgUnit(ctx context.Context, profile config.Profile, path string) (OrgUnitInfo, error)
	CreateOrgUnit(ctx context.Context, profile config.Profile, create OrgUnitCreate) (OrgUnitInfo, error)
	UpdateOrgUnit(ctx context.Context, profile config.Profile, path string, update OrgUnitUpdate) (OrgUnitInfo, error)
	DeleteOrgUnit(ctx context.Context, profile config.Profile, path string) error
}

type UserListOptions struct {
	Limit       int64
	FetchAll    bool
	Domain      string
	Query       string
	OrderBy     string
	SortOrder   string
	ShowDeleted bool
}

type GroupListOptions struct {
	Limit     int64
	FetchAll  bool
	Domain    string
	Query     string
	UserKey   string
	OrderBy   string
	SortOrder string
}

type UserUpdate struct {
	GivenName                  string
	FamilyName                 string
	OrgUnitPath                string
	RecoveryEmail              *string
	RecoveryPhone              *string
	ChangePasswordAtNextLogin  *bool
	Archived                   *bool
	IncludeInGlobalAddressList *bool
	Phones                     *any
	Addresses                  *any
	Organizations              *any
	Locations                  *any
	Relations                  *any
	ExternalIDs                *any
}

type UserCreate struct {
	PrimaryEmail              string
	GivenName                 string
	FamilyName                string
	Password                  string
	OrgUnitPath               string
	ChangePasswordAtNextLogin bool
}

type AdminDirectoryClient struct{}

func (AdminDirectoryClient) DomainInfo(ctx context.Context, profile config.Profile) (DomainInfo, error) {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return DomainInfo{}, err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return DomainInfo{}, fmt.Errorf("create Admin SDK client: %w", err)
	}
	customer, err := svc.Customers.Get("my_customer").Context(ctx).Do()
	if err != nil {
		return DomainInfo{}, fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	return domainInfo(customer), nil
}

func domainInfo(customer *admin.Customer) DomainInfo {
	return DomainInfo{
		CustomerID:           customer.Id,
		PrimaryDomain:        customer.CustomerDomain,
		AlternateEmail:       customer.AlternateEmail,
		CustomerCreationTime: customer.CustomerCreationTime,
		Language:             customer.Language,
		PhoneNumber:          customer.PhoneNumber,
		PostalAddress:        customer.PostalAddress,
		Etag:                 customer.Etag,
		Kind:                 customer.Kind,
	}
}

func (AdminDirectoryClient) Domains(ctx context.Context, profile config.Profile) ([]WorkspaceDomainInfo, error) {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return nil, err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("create Admin SDK client: %w", err)
	}
	resp, err := svc.Domains.List("my_customer").Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	domains := make([]WorkspaceDomainInfo, 0, len(resp.Domains))
	for _, domain := range resp.Domains {
		domains = append(domains, workspaceDomainInfo(domain))
	}
	return domains, nil
}

func (AdminDirectoryClient) Domain(ctx context.Context, profile config.Profile, domainName string) (WorkspaceDomainInfo, error) {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return WorkspaceDomainInfo{}, err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return WorkspaceDomainInfo{}, fmt.Errorf("create Admin SDK client: %w", err)
	}
	domain, err := svc.Domains.Get("my_customer", domainName).Context(ctx).Do()
	if err != nil {
		return WorkspaceDomainInfo{}, fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	return workspaceDomainInfo(domain), nil
}

func (AdminDirectoryClient) CreateDomain(ctx context.Context, profile config.Profile, domainName string) (WorkspaceDomainInfo, error) {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return WorkspaceDomainInfo{}, err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return WorkspaceDomainInfo{}, fmt.Errorf("create Admin SDK client: %w", err)
	}
	domain, err := svc.Domains.Insert("my_customer", &admin.Domains{DomainName: domainName}).Context(ctx).Do()
	if err != nil {
		return WorkspaceDomainInfo{}, fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	return workspaceDomainInfo(domain), nil
}

func (AdminDirectoryClient) DeleteDomain(ctx context.Context, profile config.Profile, domainName string) error {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return fmt.Errorf("create Admin SDK client: %w", err)
	}
	if err := svc.Domains.Delete("my_customer", domainName).Context(ctx).Do(); err != nil {
		return fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	return nil
}

func workspaceDomainInfo(domain *admin.Domains) WorkspaceDomainInfo {
	if domain == nil {
		return WorkspaceDomainInfo{}
	}
	aliases := make([]DomainAliasInfo, 0, len(domain.DomainAliases))
	for _, alias := range domain.DomainAliases {
		aliases = append(aliases, domainAliasInfo(alias))
	}
	return WorkspaceDomainInfo{
		DomainName:    domain.DomainName,
		IsPrimary:     domain.IsPrimary,
		Verified:      domain.Verified,
		CreationTime:  domain.CreationTime,
		DomainAliases: aliases,
		Etag:          domain.Etag,
		Kind:          domain.Kind,
	}
}

func (AdminDirectoryClient) DomainAliases(ctx context.Context, profile config.Profile) ([]DomainAliasInfo, error) {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return nil, err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("create Admin SDK client: %w", err)
	}
	resp, err := svc.DomainAliases.List("my_customer").Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	aliases := make([]DomainAliasInfo, 0, len(resp.DomainAliases))
	for _, alias := range resp.DomainAliases {
		aliases = append(aliases, domainAliasInfo(alias))
	}
	return aliases, nil
}

func (AdminDirectoryClient) DomainAlias(ctx context.Context, profile config.Profile, domainAliasName string) (DomainAliasInfo, error) {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return DomainAliasInfo{}, err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return DomainAliasInfo{}, fmt.Errorf("create Admin SDK client: %w", err)
	}
	alias, err := svc.DomainAliases.Get("my_customer", domainAliasName).Context(ctx).Do()
	if err != nil {
		return DomainAliasInfo{}, fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	return domainAliasInfo(alias), nil
}

func (AdminDirectoryClient) CreateDomainAlias(ctx context.Context, profile config.Profile, parentDomainName string, domainAliasName string) (DomainAliasInfo, error) {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return DomainAliasInfo{}, err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return DomainAliasInfo{}, fmt.Errorf("create Admin SDK client: %w", err)
	}
	alias, err := svc.DomainAliases.Insert("my_customer", &admin.DomainAlias{
		DomainAliasName:  domainAliasName,
		ParentDomainName: parentDomainName,
	}).Context(ctx).Do()
	if err != nil {
		return DomainAliasInfo{}, fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	return domainAliasInfo(alias), nil
}

func (AdminDirectoryClient) DeleteDomainAlias(ctx context.Context, profile config.Profile, domainAliasName string) error {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return fmt.Errorf("create Admin SDK client: %w", err)
	}
	if err := svc.DomainAliases.Delete("my_customer", domainAliasName).Context(ctx).Do(); err != nil {
		return fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	return nil
}

func domainAliasInfo(alias *admin.DomainAlias) DomainAliasInfo {
	if alias == nil {
		return DomainAliasInfo{}
	}
	return DomainAliasInfo{
		DomainAliasName:  alias.DomainAliasName,
		ParentDomainName: alias.ParentDomainName,
		Verified:         alias.Verified,
		CreationTime:     alias.CreationTime,
		Etag:             alias.Etag,
		Kind:             alias.Kind,
	}
}

func (AdminDirectoryClient) Users(ctx context.Context, profile config.Profile, opts UserListOptions) ([]UserInfo, error) {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return nil, err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("create Admin SDK client: %w", err)
	}
	if !opts.FetchAll && opts.Limit <= 0 {
		opts.Limit = 100
	}
	if !opts.FetchAll && opts.Limit > 500 {
		opts.Limit = 500
	}
	domain := strings.TrimSpace(opts.Domain)
	if domain == "" {
		domain = profile.Domain
	}
	users, err := collectUserPages(opts, func(pageToken string, pageSize int64) (*admin.Users, error) {
		call := svc.Users.List().
			Context(ctx).
			MaxResults(pageSize).
			Projection("basic")
		if domain != "" {
			call.Domain(domain)
		} else {
			call.Customer("my_customer")
		}
		if opts.ShowDeleted {
			call.ShowDeleted("true")
		}
		if opts.Query != "" {
			call.Query(opts.Query)
		}
		orderBy := opts.OrderBy
		if orderBy == "" {
			orderBy = "email"
		}
		if orderBy != "" {
			call.OrderBy(orderBy)
		}
		if opts.SortOrder != "" {
			call.SortOrder(opts.SortOrder)
		}
		if pageToken != "" {
			call.PageToken(pageToken)
		}
		return call.Do()
	})
	if err != nil {
		return nil, fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	return users, nil
}

func (AdminDirectoryClient) User(ctx context.Context, profile config.Profile, email string) (UserInfo, error) {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return UserInfo{}, err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return UserInfo{}, fmt.Errorf("create Admin SDK client: %w", err)
	}
	user, err := svc.Users.Get(email).Context(ctx).Projection("full").Do()
	if err != nil {
		return UserInfo{}, fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	return userInfo(user), nil
}

func (AdminDirectoryClient) CreateUser(ctx context.Context, profile config.Profile, create UserCreate) (UserInfo, error) {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return UserInfo{}, err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return UserInfo{}, fmt.Errorf("create Admin SDK client: %w", err)
	}
	user := &admin.User{
		PrimaryEmail: create.PrimaryEmail,
		Name: &admin.UserName{
			GivenName:  create.GivenName,
			FamilyName: create.FamilyName,
		},
		Password:                  create.Password,
		OrgUnitPath:               create.OrgUnitPath,
		ChangePasswordAtNextLogin: create.ChangePasswordAtNextLogin,
	}
	if create.OrgUnitPath != "" {
		user.ForceSendFields = append(user.ForceSendFields, "OrgUnitPath")
	}
	if create.ChangePasswordAtNextLogin {
		user.ForceSendFields = append(user.ForceSendFields, "ChangePasswordAtNextLogin")
	}
	created, err := svc.Users.Insert(user).Context(ctx).Do()
	if err != nil {
		return UserInfo{}, fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	return userInfo(created), nil
}

func (AdminDirectoryClient) UpdateUser(ctx context.Context, profile config.Profile, email string, update UserUpdate) (UserInfo, error) {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return UserInfo{}, err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return UserInfo{}, fmt.Errorf("create Admin SDK client: %w", err)
	}
	patch := &admin.User{}
	if update.GivenName != "" || update.FamilyName != "" {
		patch.Name = &admin.UserName{
			GivenName:  update.GivenName,
			FamilyName: update.FamilyName,
		}
		if update.GivenName != "" {
			patch.Name.ForceSendFields = append(patch.Name.ForceSendFields, "GivenName")
		}
		if update.FamilyName != "" {
			patch.Name.ForceSendFields = append(patch.Name.ForceSendFields, "FamilyName")
		}
	}
	if update.OrgUnitPath != "" {
		patch.OrgUnitPath = update.OrgUnitPath
		patch.ForceSendFields = append(patch.ForceSendFields, "OrgUnitPath")
	}
	if update.RecoveryEmail != nil {
		patch.RecoveryEmail = *update.RecoveryEmail
		patch.ForceSendFields = append(patch.ForceSendFields, "RecoveryEmail")
	}
	if update.RecoveryPhone != nil {
		patch.RecoveryPhone = *update.RecoveryPhone
		patch.ForceSendFields = append(patch.ForceSendFields, "RecoveryPhone")
	}
	if update.ChangePasswordAtNextLogin != nil {
		patch.ChangePasswordAtNextLogin = *update.ChangePasswordAtNextLogin
		patch.ForceSendFields = append(patch.ForceSendFields, "ChangePasswordAtNextLogin")
	}
	if update.Archived != nil {
		patch.Archived = *update.Archived
		patch.ForceSendFields = append(patch.ForceSendFields, "Archived")
	}
	if update.IncludeInGlobalAddressList != nil {
		patch.IncludeInGlobalAddressList = *update.IncludeInGlobalAddressList
		patch.ForceSendFields = append(patch.ForceSendFields, "IncludeInGlobalAddressList")
	}
	if update.Phones != nil {
		patch.Phones = *update.Phones
		patch.ForceSendFields = append(patch.ForceSendFields, "Phones")
	}
	if update.Addresses != nil {
		patch.Addresses = *update.Addresses
		patch.ForceSendFields = append(patch.ForceSendFields, "Addresses")
	}
	if update.Organizations != nil {
		patch.Organizations = *update.Organizations
		patch.ForceSendFields = append(patch.ForceSendFields, "Organizations")
	}
	if update.Locations != nil {
		patch.Locations = *update.Locations
		patch.ForceSendFields = append(patch.ForceSendFields, "Locations")
	}
	if update.Relations != nil {
		patch.Relations = *update.Relations
		patch.ForceSendFields = append(patch.ForceSendFields, "Relations")
	}
	if update.ExternalIDs != nil {
		patch.ExternalIds = *update.ExternalIDs
		patch.ForceSendFields = append(patch.ForceSendFields, "ExternalIds")
	}
	user, err := svc.Users.Patch(email, patch).Context(ctx).Do()
	if err != nil {
		return UserInfo{}, fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	return userInfo(user), nil
}

func (AdminDirectoryClient) DeleteUser(ctx context.Context, profile config.Profile, email string) error {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return fmt.Errorf("create Admin SDK client: %w", err)
	}
	if err := svc.Users.Delete(email).Context(ctx).Do(); err != nil {
		return fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	return nil
}

func (AdminDirectoryClient) SetUserAdmin(ctx context.Context, profile config.Profile, email string, adminStatus bool) error {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return fmt.Errorf("create Admin SDK client: %w", err)
	}
	if err := svc.Users.MakeAdmin(email, &admin.UserMakeAdmin{
		Status:          adminStatus,
		ForceSendFields: []string{"Status"},
	}).Context(ctx).Do(); err != nil {
		return fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	return nil
}

func (AdminDirectoryClient) SetUserSuspended(ctx context.Context, profile config.Profile, email string, suspended bool) (UserInfo, error) {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return UserInfo{}, err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return UserInfo{}, fmt.Errorf("create Admin SDK client: %w", err)
	}
	user, err := svc.Users.Patch(email, &admin.User{
		Suspended:       suspended,
		ForceSendFields: []string{"Suspended"},
	}).Context(ctx).Do()
	if err != nil {
		return UserInfo{}, fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	return userInfo(user), nil
}

func (AdminDirectoryClient) UserAliases(ctx context.Context, profile config.Profile, userKey string) ([]AliasInfo, error) {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return nil, err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("create Admin SDK client: %w", err)
	}
	resp, err := svc.Users.Aliases.List(userKey).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	return aliasInfos(resp.Aliases), nil
}

func (AdminDirectoryClient) CreateUserAlias(ctx context.Context, profile config.Profile, userKey string, alias string) (AliasInfo, error) {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return AliasInfo{}, err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return AliasInfo{}, fmt.Errorf("create Admin SDK client: %w", err)
	}
	created, err := svc.Users.Aliases.Insert(userKey, &admin.Alias{Alias: alias}).Context(ctx).Do()
	if err != nil {
		return AliasInfo{}, fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	return aliasInfo(created), nil
}

func (AdminDirectoryClient) DeleteUserAlias(ctx context.Context, profile config.Profile, userKey string, alias string) error {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return fmt.Errorf("create Admin SDK client: %w", err)
	}
	if err := svc.Users.Aliases.Delete(userKey, alias).Context(ctx).Do(); err != nil {
		return fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	return nil
}

func userInfo(user *admin.User) UserInfo {
	info := UserInfo{
		ID:                         user.Id,
		CustomerID:                 user.CustomerId,
		PrimaryEmail:               user.PrimaryEmail,
		Aliases:                    append([]string(nil), user.Aliases...),
		NonEditableAliases:         append([]string(nil), user.NonEditableAliases...),
		Suspended:                  user.Suspended,
		SuspensionReason:           user.SuspensionReason,
		IsArchived:                 user.Archived,
		OrgUnitPath:                user.OrgUnitPath,
		IsAdmin:                    user.IsAdmin,
		IsDelegatedAdmin:           user.IsDelegatedAdmin,
		IsEnrolledIn2SV:            user.IsEnrolledIn2Sv,
		IsEnforcedIn2SV:            user.IsEnforcedIn2Sv,
		IsMailboxSetup:             user.IsMailboxSetup,
		IsGuestUser:                user.IsGuestUser,
		IncludeInGlobalAddressList: user.IncludeInGlobalAddressList,
		AgreedToTerms:              user.AgreedToTerms,
		ChangePasswordAtNextLogin:  user.ChangePasswordAtNextLogin,
		IPWhitelisted:              user.IpWhitelisted,
		CreationTime:               user.CreationTime,
		LastLoginTime:              user.LastLoginTime,
		DeletionTime:               user.DeletionTime,
		RecoveryEmail:              user.RecoveryEmail,
		RecoveryPhone:              user.RecoveryPhone,
		ThumbnailPhotoURL:          user.ThumbnailPhotoUrl,
		ThumbnailPhotoEtag:         user.ThumbnailPhotoEtag,
		Emails:                     user.Emails,
		Phones:                     user.Phones,
		Addresses:                  user.Addresses,
		Organizations:              user.Organizations,
		Relations:                  user.Relations,
		ExternalIDs:                user.ExternalIds,
		Locations:                  user.Locations,
	}
	if user.Name != nil {
		info.Name = strings.TrimSpace(user.Name.FullName)
		info.GivenName = strings.TrimSpace(user.Name.GivenName)
		info.FamilyName = strings.TrimSpace(user.Name.FamilyName)
		if info.Name == "" {
			info.Name = strings.TrimSpace(info.GivenName + " " + info.FamilyName)
		}
	}
	if user.CustomSchemas != nil {
		info.CustomSchemas = make(map[string]any, len(user.CustomSchemas))
		for name, schema := range user.CustomSchemas {
			info.CustomSchemas[name] = schema
		}
	}
	return info
}

func aliasInfos(raw []interface{}) []AliasInfo {
	aliases := make([]AliasInfo, 0, len(raw))
	for _, alias := range raw {
		aliases = append(aliases, aliasInfoFromAny(alias))
	}
	return aliases
}

func aliasInfo(alias *admin.Alias) AliasInfo {
	if alias == nil {
		return AliasInfo{}
	}
	return AliasInfo{
		Alias:        alias.Alias,
		PrimaryEmail: alias.PrimaryEmail,
		ID:           alias.Id,
		Etag:         alias.Etag,
		Kind:         alias.Kind,
	}
}

func aliasInfoFromAny(raw any) AliasInfo {
	if alias, ok := raw.(*admin.Alias); ok {
		return aliasInfo(alias)
	}
	if alias, ok := raw.(admin.Alias); ok {
		return aliasInfo(&alias)
	}
	fields, ok := raw.(map[string]any)
	if !ok {
		return AliasInfo{}
	}
	return AliasInfo{
		Alias:        stringValue(fields["alias"]),
		PrimaryEmail: stringValue(fields["primaryEmail"]),
		ID:           stringValue(fields["id"]),
		Etag:         stringValue(fields["etag"]),
		Kind:         stringValue(fields["kind"]),
	}
}

func stringValue(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func (AdminDirectoryClient) Groups(ctx context.Context, profile config.Profile, opts GroupListOptions) ([]GroupInfo, error) {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return nil, err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("create Admin SDK client: %w", err)
	}
	if !opts.FetchAll && opts.Limit <= 0 {
		opts.Limit = 100
	}
	if !opts.FetchAll && opts.Limit > 200 {
		opts.Limit = 200
	}
	domain := strings.TrimSpace(opts.Domain)
	if domain == "" {
		domain = profile.Domain
	}
	groups, err := collectGroupPages(opts, func(pageToken string, pageSize int64) (*admin.Groups, error) {
		call := svc.Groups.List().
			Context(ctx).
			MaxResults(pageSize)
		if opts.UserKey != "" {
			call.UserKey(opts.UserKey)
		} else if domain != "" {
			call.Domain(domain)
		} else {
			call.Customer("my_customer")
		}
		if opts.Query != "" {
			call.Query(opts.Query)
		}
		orderBy := opts.OrderBy
		if orderBy == "" {
			orderBy = "email"
		}
		if orderBy != "" {
			call.OrderBy(orderBy)
		}
		if opts.SortOrder != "" {
			call.SortOrder(opts.SortOrder)
		}
		if pageToken != "" {
			call.PageToken(pageToken)
		}
		return call.Do()
	})
	if err != nil {
		return nil, fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	return groups, nil
}

func collectUserPages(opts UserListOptions, fetch func(pageToken string, pageSize int64) (*admin.Users, error)) ([]UserInfo, error) {
	const maxPageSize int64 = 500
	target := opts.Limit
	if opts.FetchAll {
		target = 0
	}
	users := []UserInfo{}
	pageToken := ""
	for {
		pageSize := maxPageSize
		if target > 0 && target-int64(len(users)) < pageSize {
			pageSize = target - int64(len(users))
		}
		if pageSize <= 0 {
			break
		}
		resp, err := fetch(pageToken, pageSize)
		if err != nil {
			return nil, err
		}
		for _, user := range resp.Users {
			users = append(users, userInfo(user))
			if target > 0 && int64(len(users)) >= target {
				return users, nil
			}
		}
		if resp.NextPageToken == "" {
			return users, nil
		}
		pageToken = resp.NextPageToken
	}
	return users, nil
}

func collectGroupPages(opts GroupListOptions, fetch func(pageToken string, pageSize int64) (*admin.Groups, error)) ([]GroupInfo, error) {
	const maxPageSize int64 = 200
	target := opts.Limit
	if opts.FetchAll {
		target = 0
	}
	groups := []GroupInfo{}
	pageToken := ""
	for {
		pageSize := maxPageSize
		if target > 0 && target-int64(len(groups)) < pageSize {
			pageSize = target - int64(len(groups))
		}
		if pageSize <= 0 {
			break
		}
		resp, err := fetch(pageToken, pageSize)
		if err != nil {
			return nil, err
		}
		for _, group := range resp.Groups {
			groups = append(groups, groupInfo(group))
			if target > 0 && int64(len(groups)) >= target {
				return groups, nil
			}
		}
		if resp.NextPageToken == "" {
			return groups, nil
		}
		pageToken = resp.NextPageToken
	}
	return groups, nil
}

func (AdminDirectoryClient) Group(ctx context.Context, profile config.Profile, email string) (GroupInfo, error) {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return GroupInfo{}, err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return GroupInfo{}, fmt.Errorf("create Admin SDK client: %w", err)
	}
	group, err := svc.Groups.Get(email).Context(ctx).Do()
	if err != nil {
		return GroupInfo{}, fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	return groupInfo(group), nil
}

func (AdminDirectoryClient) CreateGroup(ctx context.Context, profile config.Profile, group GroupInfo) (GroupInfo, error) {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return GroupInfo{}, err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return GroupInfo{}, fmt.Errorf("create Admin SDK client: %w", err)
	}
	created, err := svc.Groups.Insert(&admin.Group{
		Email:       group.Email,
		Name:        group.Name,
		Description: group.Description,
	}).Context(ctx).Do()
	if err != nil {
		return GroupInfo{}, fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	return groupInfo(created), nil
}

func (AdminDirectoryClient) UpdateGroup(ctx context.Context, profile config.Profile, email string, group GroupInfo) (GroupInfo, error) {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return GroupInfo{}, err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return GroupInfo{}, fmt.Errorf("create Admin SDK client: %w", err)
	}
	patch := &admin.Group{
		Email:       group.Email,
		Name:        group.Name,
		Description: group.Description,
	}
	if group.Email != "" {
		patch.ForceSendFields = append(patch.ForceSendFields, "Email")
	}
	if group.Name != "" {
		patch.ForceSendFields = append(patch.ForceSendFields, "Name")
	}
	if group.Description != "" {
		patch.ForceSendFields = append(patch.ForceSendFields, "Description")
	}
	updated, err := svc.Groups.Patch(email, patch).Context(ctx).Do()
	if err != nil {
		return GroupInfo{}, fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	return groupInfo(updated), nil
}

func (AdminDirectoryClient) DeleteGroup(ctx context.Context, profile config.Profile, email string) error {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return fmt.Errorf("create Admin SDK client: %w", err)
	}
	if err := svc.Groups.Delete(email).Context(ctx).Do(); err != nil {
		return fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	return nil
}

func (AdminDirectoryClient) GroupAliases(ctx context.Context, profile config.Profile, groupKey string) ([]AliasInfo, error) {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return nil, err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("create Admin SDK client: %w", err)
	}
	resp, err := svc.Groups.Aliases.List(groupKey).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	return aliasInfos(resp.Aliases), nil
}

func (AdminDirectoryClient) CreateGroupAlias(ctx context.Context, profile config.Profile, groupKey string, alias string) (AliasInfo, error) {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return AliasInfo{}, err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return AliasInfo{}, fmt.Errorf("create Admin SDK client: %w", err)
	}
	created, err := svc.Groups.Aliases.Insert(groupKey, &admin.Alias{Alias: alias}).Context(ctx).Do()
	if err != nil {
		return AliasInfo{}, fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	return aliasInfo(created), nil
}

func (AdminDirectoryClient) DeleteGroupAlias(ctx context.Context, profile config.Profile, groupKey string, alias string) error {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return fmt.Errorf("create Admin SDK client: %w", err)
	}
	if err := svc.Groups.Aliases.Delete(groupKey, alias).Context(ctx).Do(); err != nil {
		return fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	return nil
}

func groupInfo(group *admin.Group) GroupInfo {
	return GroupInfo{
		Email:              group.Email,
		ID:                 group.Id,
		Name:               group.Name,
		Description:        group.Description,
		DirectMembersCount: group.DirectMembersCount,
		AdminCreated:       group.AdminCreated,
		Aliases:            append([]string(nil), group.Aliases...),
		NonEditableAliases: append([]string(nil), group.NonEditableAliases...),
		Etag:               group.Etag,
		Kind:               group.Kind,
	}
}

func (AdminDirectoryClient) GroupMembers(ctx context.Context, profile config.Profile, groupEmail string, limit int64) ([]MemberInfo, error) {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return nil, err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("create Admin SDK client: %w", err)
	}
	fetchAll := limit == 0
	if !fetchAll && limit < 0 {
		limit = 100
	}
	if !fetchAll && limit > 200 {
		limit = 200
	}
	members, err := collectMemberPages(limit, fetchAll, func(pageToken string, pageSize int64) (*admin.Members, error) {
		call := svc.Members.List(groupEmail).
			Context(ctx).
			MaxResults(pageSize)
		if pageToken != "" {
			call.PageToken(pageToken)
		}
		return call.Do()
	})
	if err != nil {
		return nil, fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	return members, nil
}

func (AdminDirectoryClient) GroupMember(ctx context.Context, profile config.Profile, groupEmail string, memberEmail string) (MemberInfo, error) {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return MemberInfo{}, err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return MemberInfo{}, fmt.Errorf("create Admin SDK client: %w", err)
	}
	member, err := svc.Members.Get(groupEmail, memberEmail).Context(ctx).Do()
	if err != nil {
		return MemberInfo{}, fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	return memberInfo(member), nil
}

func (AdminDirectoryClient) AddGroupMember(ctx context.Context, profile config.Profile, groupEmail string, memberEmail string, role string) (MemberInfo, error) {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return MemberInfo{}, err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return MemberInfo{}, fmt.Errorf("create Admin SDK client: %w", err)
	}
	member, err := svc.Members.Insert(groupEmail, &admin.Member{
		Email: memberEmail,
		Role:  role,
	}).Context(ctx).Do()
	if err != nil {
		return MemberInfo{}, fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	return memberInfo(member), nil
}

func (AdminDirectoryClient) UpdateGroupMember(ctx context.Context, profile config.Profile, groupEmail string, memberEmail string, role string) (MemberInfo, error) {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return MemberInfo{}, err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return MemberInfo{}, fmt.Errorf("create Admin SDK client: %w", err)
	}
	member, err := svc.Members.Update(groupEmail, memberEmail, &admin.Member{Role: role}).Context(ctx).Do()
	if err != nil {
		return MemberInfo{}, fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	return memberInfo(member), nil
}

func (AdminDirectoryClient) RemoveGroupMember(ctx context.Context, profile config.Profile, groupEmail string, memberEmail string) error {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return fmt.Errorf("create Admin SDK client: %w", err)
	}
	if err := svc.Members.Delete(groupEmail, memberEmail).Context(ctx).Do(); err != nil {
		return fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	return nil
}

func collectMemberPages(limit int64, fetchAll bool, fetch func(pageToken string, pageSize int64) (*admin.Members, error)) ([]MemberInfo, error) {
	const maxPageSize int64 = 200
	target := limit
	if fetchAll {
		target = 0
	}
	if target < 0 {
		target = 100
	}
	members := []MemberInfo{}
	pageToken := ""
	for {
		pageSize := maxPageSize
		if target > 0 && target-int64(len(members)) < pageSize {
			pageSize = target - int64(len(members))
		}
		if pageSize <= 0 {
			break
		}
		resp, err := fetch(pageToken, pageSize)
		if err != nil {
			return nil, err
		}
		for _, member := range resp.Members {
			members = append(members, memberInfo(member))
			if target > 0 && int64(len(members)) >= target {
				return members, nil
			}
		}
		if resp.NextPageToken == "" {
			return members, nil
		}
		pageToken = resp.NextPageToken
	}
	return members, nil
}

func memberInfo(member *admin.Member) MemberInfo {
	return MemberInfo{
		Email:            member.Email,
		ID:               member.Id,
		Role:             member.Role,
		Type:             member.Type,
		Status:           member.Status,
		DeliverySettings: member.DeliverySettings,
		Etag:             member.Etag,
		Kind:             member.Kind,
	}
}

func (AdminDirectoryClient) OrgUnits(ctx context.Context, profile config.Profile) ([]OrgUnitInfo, error) {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return nil, err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("create Admin SDK client: %w", err)
	}
	resp, err := svc.Orgunits.List("my_customer").
		Context(ctx).
		OrgUnitPath("/").
		Type("allIncludingParent").
		Do()
	if err != nil {
		return nil, fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	ous := make([]OrgUnitInfo, 0, len(resp.OrganizationUnits))
	for _, ou := range resp.OrganizationUnits {
		ous = append(ous, orgUnitInfo(ou))
	}
	return ous, nil
}

func (AdminDirectoryClient) OrgUnit(ctx context.Context, profile config.Profile, path string) (OrgUnitInfo, error) {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return OrgUnitInfo{}, err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return OrgUnitInfo{}, fmt.Errorf("create Admin SDK client: %w", err)
	}
	ou, err := svc.Orgunits.Get("my_customer", path).Context(ctx).Do()
	if err != nil {
		return OrgUnitInfo{}, fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	return orgUnitInfo(ou), nil
}

func (AdminDirectoryClient) CreateOrgUnit(ctx context.Context, profile config.Profile, create OrgUnitCreate) (OrgUnitInfo, error) {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return OrgUnitInfo{}, err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return OrgUnitInfo{}, fmt.Errorf("create Admin SDK client: %w", err)
	}
	ou, err := svc.Orgunits.Insert("my_customer", &admin.OrgUnit{
		Name:              create.Name,
		ParentOrgUnitPath: create.ParentOrgUnitPath,
		Description:       create.Description,
	}).Context(ctx).Do()
	if err != nil {
		return OrgUnitInfo{}, fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	return orgUnitInfo(ou), nil
}

func (AdminDirectoryClient) UpdateOrgUnit(ctx context.Context, profile config.Profile, path string, update OrgUnitUpdate) (OrgUnitInfo, error) {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return OrgUnitInfo{}, err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return OrgUnitInfo{}, fmt.Errorf("create Admin SDK client: %w", err)
	}
	patch := &admin.OrgUnit{
		Name:              update.Name,
		ParentOrgUnitPath: update.ParentOrgUnitPath,
		Description:       update.Description,
	}
	if update.Name != "" {
		patch.ForceSendFields = append(patch.ForceSendFields, "Name")
	}
	if update.ParentOrgUnitPath != "" {
		patch.ForceSendFields = append(patch.ForceSendFields, "ParentOrgUnitPath")
	}
	if update.Description != "" {
		patch.ForceSendFields = append(patch.ForceSendFields, "Description")
	}
	ou, err := svc.Orgunits.Patch("my_customer", path, patch).Context(ctx).Do()
	if err != nil {
		return OrgUnitInfo{}, fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	return orgUnitInfo(ou), nil
}

func (AdminDirectoryClient) DeleteOrgUnit(ctx context.Context, profile config.Profile, path string) error {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return fmt.Errorf("create Admin SDK client: %w", err)
	}
	if err := svc.Orgunits.Delete("my_customer", path).Context(ctx).Do(); err != nil {
		return fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	return nil
}

func orgUnitInfo(ou *admin.OrgUnit) OrgUnitInfo {
	return OrgUnitInfo{
		Name:              ou.Name,
		OrgUnitID:         ou.OrgUnitId,
		OrgUnitPath:       ou.OrgUnitPath,
		ParentOrgUnitID:   ou.ParentOrgUnitId,
		ParentOrgUnitPath: ou.ParentOrgUnitPath,
		Description:       ou.Description,
		BlockInheritance:  ou.BlockInheritance,
		Etag:              ou.Etag,
		Kind:              ou.Kind,
	}
}
