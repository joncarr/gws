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
	CustomerID         string `json:"customer_id"`
	PrimaryDomain      string `json:"primary_domain"`
	VerifiedDomainName string `json:"verified_domain_name,omitempty"`
}

type UserInfo struct {
	PrimaryEmail               string `json:"primary_email"`
	Name                       string `json:"name,omitempty"`
	Suspended                  bool   `json:"suspended"`
	OrgUnitPath                string `json:"org_unit_path,omitempty"`
	IsAdmin                    bool   `json:"is_admin"`
	IsDelegatedAdmin           bool   `json:"is_delegated_admin"`
	IncludeInGlobalAddressList bool   `json:"include_in_global_address_list"`
	CreationTime               string `json:"creation_time,omitempty"`
	LastLoginTime              string `json:"last_login_time,omitempty"`
}

type GroupInfo struct {
	Email              string   `json:"email"`
	Name               string   `json:"name,omitempty"`
	Description        string   `json:"description,omitempty"`
	DirectMembersCount int64    `json:"direct_members_count"`
	AdminCreated       bool     `json:"admin_created"`
	Aliases            []string `json:"aliases,omitempty"`
	NonEditableAliases []string `json:"non_editable_aliases,omitempty"`
}

type MemberInfo struct {
	Email  string `json:"email"`
	Role   string `json:"role,omitempty"`
	Type   string `json:"type,omitempty"`
	Status string `json:"status,omitempty"`
}

type OrgUnitInfo struct {
	Name              string `json:"name"`
	OrgUnitID         string `json:"org_unit_id,omitempty"`
	OrgUnitPath       string `json:"org_unit_path"`
	ParentOrgUnitID   string `json:"parent_org_unit_id,omitempty"`
	ParentOrgUnitPath string `json:"parent_org_unit_path,omitempty"`
	Description       string `json:"description,omitempty"`
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
	Users(ctx context.Context, profile config.Profile, limit int64) ([]UserInfo, error)
	User(ctx context.Context, profile config.Profile, email string) (UserInfo, error)
	CreateUser(ctx context.Context, profile config.Profile, create UserCreate) (UserInfo, error)
	UpdateUser(ctx context.Context, profile config.Profile, email string, update UserUpdate) (UserInfo, error)
	SetUserSuspended(ctx context.Context, profile config.Profile, email string, suspended bool) (UserInfo, error)
	Groups(ctx context.Context, profile config.Profile, limit int64) ([]GroupInfo, error)
	Group(ctx context.Context, profile config.Profile, email string) (GroupInfo, error)
	CreateGroup(ctx context.Context, profile config.Profile, group GroupInfo) (GroupInfo, error)
	UpdateGroup(ctx context.Context, profile config.Profile, email string, group GroupInfo) (GroupInfo, error)
	GroupMembers(ctx context.Context, profile config.Profile, groupEmail string, limit int64) ([]MemberInfo, error)
	AddGroupMember(ctx context.Context, profile config.Profile, groupEmail string, memberEmail string, role string) (MemberInfo, error)
	RemoveGroupMember(ctx context.Context, profile config.Profile, groupEmail string, memberEmail string) error
	OrgUnits(ctx context.Context, profile config.Profile) ([]OrgUnitInfo, error)
	OrgUnit(ctx context.Context, profile config.Profile, path string) (OrgUnitInfo, error)
	CreateOrgUnit(ctx context.Context, profile config.Profile, create OrgUnitCreate) (OrgUnitInfo, error)
	UpdateOrgUnit(ctx context.Context, profile config.Profile, path string, update OrgUnitUpdate) (OrgUnitInfo, error)
}

type UserUpdate struct {
	GivenName   string
	FamilyName  string
	OrgUnitPath string
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
	info := DomainInfo{
		CustomerID:    customer.Id,
		PrimaryDomain: customer.CustomerDomain,
	}
	return info, nil
}

func (AdminDirectoryClient) Users(ctx context.Context, profile config.Profile, limit int64) ([]UserInfo, error) {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return nil, err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("create Admin SDK client: %w", err)
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	call := svc.Users.List().
		Context(ctx).
		Domain(profile.Domain).
		MaxResults(limit).
		OrderBy("email").
		Projection("basic")
	resp, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	users := make([]UserInfo, 0, len(resp.Users))
	for _, user := range resp.Users {
		users = append(users, userInfo(user))
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
	user, err := svc.Users.Patch(email, patch).Context(ctx).Do()
	if err != nil {
		return UserInfo{}, fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	return userInfo(user), nil
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

func userInfo(user *admin.User) UserInfo {
	info := UserInfo{
		PrimaryEmail:               user.PrimaryEmail,
		Suspended:                  user.Suspended,
		OrgUnitPath:                user.OrgUnitPath,
		IsAdmin:                    user.IsAdmin,
		IsDelegatedAdmin:           user.IsDelegatedAdmin,
		IncludeInGlobalAddressList: user.IncludeInGlobalAddressList,
		CreationTime:               user.CreationTime,
		LastLoginTime:              user.LastLoginTime,
	}
	if user.Name != nil {
		info.Name = strings.TrimSpace(user.Name.FullName)
	}
	return info
}

func (AdminDirectoryClient) Groups(ctx context.Context, profile config.Profile, limit int64) ([]GroupInfo, error) {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return nil, err
	}
	svc, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("create Admin SDK client: %w", err)
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 200 {
		limit = 200
	}
	resp, err := svc.Groups.List().
		Context(ctx).
		Domain(profile.Domain).
		MaxResults(limit).
		OrderBy("email").
		Do()
	if err != nil {
		return nil, fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	groups := make([]GroupInfo, 0, len(resp.Groups))
	for _, group := range resp.Groups {
		groups = append(groups, groupInfo(group))
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
		Name:        group.Name,
		Description: group.Description,
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

func groupInfo(group *admin.Group) GroupInfo {
	return GroupInfo{
		Email:              group.Email,
		Name:               group.Name,
		Description:        group.Description,
		DirectMembersCount: group.DirectMembersCount,
		AdminCreated:       group.AdminCreated,
		Aliases:            append([]string(nil), group.Aliases...),
		NonEditableAliases: append([]string(nil), group.NonEditableAliases...),
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
	if limit <= 0 {
		limit = 100
	}
	if limit > 200 {
		limit = 200
	}
	resp, err := svc.Members.List(groupEmail).
		Context(ctx).
		MaxResults(limit).
		Do()
	if err != nil {
		return nil, fmt.Errorf("call Admin SDK Directory API: %w", err)
	}
	members := make([]MemberInfo, 0, len(resp.Members))
	for _, member := range resp.Members {
		members = append(members, memberInfo(member))
	}
	return members, nil
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

func memberInfo(member *admin.Member) MemberInfo {
	return MemberInfo{
		Email:  member.Email,
		Role:   member.Role,
		Type:   member.Type,
		Status: member.Status,
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

func orgUnitInfo(ou *admin.OrgUnit) OrgUnitInfo {
	return OrgUnitInfo{
		Name:              ou.Name,
		OrgUnitID:         ou.OrgUnitId,
		OrgUnitPath:       ou.OrgUnitPath,
		ParentOrgUnitID:   ou.ParentOrgUnitId,
		ParentOrgUnitPath: ou.ParentOrgUnitPath,
		Description:       ou.Description,
	}
}
