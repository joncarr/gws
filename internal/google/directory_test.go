package google

import (
	"fmt"
	"testing"

	admin "google.golang.org/api/admin/directory/v1"
)

func TestDomainInfoMapsDirectoryCustomerFields(t *testing.T) {
	customer := &admin.Customer{
		Id:                   "C01",
		CustomerDomain:       "example.com",
		AlternateEmail:       "owner@example.net",
		CustomerCreationTime: "2026-01-02T03:04:05.000Z",
		Language:             "en",
		PhoneNumber:          "+15555550100",
		Etag:                 "etag-1",
		Kind:                 "admin#directory#customer",
	}

	info := domainInfo(customer)
	if info.CustomerID != "C01" {
		t.Fatalf("CustomerID = %q", info.CustomerID)
	}
	if info.AlternateEmail != "owner@example.net" {
		t.Fatalf("AlternateEmail = %q", info.AlternateEmail)
	}
	if info.Kind != "admin#directory#customer" {
		t.Fatalf("Kind = %q", info.Kind)
	}
}

func TestWorkspaceDomainInfoMapsDirectoryDomainFields(t *testing.T) {
	domain := &admin.Domains{
		DomainName:   "example.com",
		IsPrimary:    true,
		Verified:     true,
		CreationTime: 12345,
		Etag:         "etag-1",
		Kind:         "admin#directory#domain",
		DomainAliases: []*admin.DomainAlias{
			{DomainAliasName: "alias.example.com", ParentDomainName: "example.com", Verified: true},
		},
	}

	info := workspaceDomainInfo(domain)
	if info.DomainName != "example.com" {
		t.Fatalf("DomainName = %q", info.DomainName)
	}
	if !info.IsPrimary || !info.Verified {
		t.Fatalf("domain booleans = primary %t verified %t", info.IsPrimary, info.Verified)
	}
	if len(info.DomainAliases) != 1 || info.DomainAliases[0].DomainAliasName != "alias.example.com" {
		t.Fatalf("DomainAliases = %#v", info.DomainAliases)
	}
}

func TestDomainAliasInfoMapsDirectoryDomainAliasFields(t *testing.T) {
	alias := &admin.DomainAlias{
		DomainAliasName:  "alias.example.com",
		ParentDomainName: "example.com",
		Verified:         true,
		CreationTime:     12345,
		Etag:             "etag-1",
		Kind:             "admin#directory#domainAlias",
	}

	info := domainAliasInfo(alias)
	if info.DomainAliasName != "alias.example.com" {
		t.Fatalf("DomainAliasName = %q", info.DomainAliasName)
	}
	if info.ParentDomainName != "example.com" {
		t.Fatalf("ParentDomainName = %q", info.ParentDomainName)
	}
	if !info.Verified {
		t.Fatal("Verified = false")
	}
}

func TestUserInfoMapsDirectoryUserFields(t *testing.T) {
	user := &admin.User{
		Id:                         "12345",
		CustomerId:                 "C01",
		PrimaryEmail:               "ada@example.com",
		Aliases:                    []string{"a@example.com"},
		NonEditableAliases:         []string{"ada@alias.example.com"},
		Suspended:                  true,
		SuspensionReason:           "ADMIN",
		Archived:                   true,
		OrgUnitPath:                "/Engineering",
		IsAdmin:                    true,
		IsDelegatedAdmin:           true,
		IsEnrolledIn2Sv:            true,
		IsEnforcedIn2Sv:            true,
		IsMailboxSetup:             true,
		IncludeInGlobalAddressList: true,
		AgreedToTerms:              true,
		ChangePasswordAtNextLogin:  true,
		IpWhitelisted:              true,
		CreationTime:               "2026-01-02T03:04:05.000Z",
		LastLoginTime:              "2026-01-03T03:04:05.000Z",
		DeletionTime:               "2026-01-04T03:04:05.000Z",
		RecoveryEmail:              "recovery@example.com",
		RecoveryPhone:              "+15555550100",
		ThumbnailPhotoUrl:          "https://example.com/photo",
		Name: &admin.UserName{
			FullName:   "Ada Lovelace",
			GivenName:  "Ada",
			FamilyName: "Lovelace",
		},
	}

	info := userInfo(user)
	if info.ID != "12345" {
		t.Fatalf("ID = %q", info.ID)
	}
	if !info.IsArchived {
		t.Fatal("IsArchived = false")
	}
	if !info.IsEnrolledIn2SV || !info.IsEnforcedIn2SV {
		t.Fatalf("2SV fields = enrolled %t enforced %t", info.IsEnrolledIn2SV, info.IsEnforcedIn2SV)
	}
	if info.GivenName != "Ada" || info.FamilyName != "Lovelace" {
		t.Fatalf("name fields = %q %q", info.GivenName, info.FamilyName)
	}
	if len(info.Aliases) != 1 || info.Aliases[0] != "a@example.com" {
		t.Fatalf("Aliases = %#v", info.Aliases)
	}
	if info.RecoveryEmail != "recovery@example.com" {
		t.Fatalf("RecoveryEmail = %q", info.RecoveryEmail)
	}
}

func TestAliasInfoMapsDirectoryAliasFields(t *testing.T) {
	alias := &admin.Alias{
		Alias:        "ada.alias@example.com",
		PrimaryEmail: "ada@example.com",
		Id:           "alias-id",
		Etag:         "etag-1",
		Kind:         "admin#directory#alias",
	}

	info := aliasInfo(alias)
	if info.Alias != "ada.alias@example.com" {
		t.Fatalf("Alias = %q", info.Alias)
	}
	if info.PrimaryEmail != "ada@example.com" {
		t.Fatalf("PrimaryEmail = %q", info.PrimaryEmail)
	}
	if info.ID != "alias-id" {
		t.Fatalf("ID = %q", info.ID)
	}
}

func TestAliasInfoMapsListAliasFields(t *testing.T) {
	info := aliasInfoFromAny(map[string]any{
		"alias":        "ada.alias@example.com",
		"primaryEmail": "ada@example.com",
		"id":           "alias-id",
		"etag":         "etag-1",
		"kind":         "admin#directory#alias",
	})
	if info.Alias != "ada.alias@example.com" {
		t.Fatalf("Alias = %q", info.Alias)
	}
	if info.PrimaryEmail != "ada@example.com" {
		t.Fatalf("PrimaryEmail = %q", info.PrimaryEmail)
	}
	if info.Kind != "admin#directory#alias" {
		t.Fatalf("Kind = %q", info.Kind)
	}
}

func TestGroupInfoMapsDirectoryGroupFields(t *testing.T) {
	group := &admin.Group{
		Email:              "eng@example.com",
		Id:                 "group-id",
		Name:               "Engineering",
		Description:        "Engineering team",
		DirectMembersCount: 12,
		AdminCreated:       true,
		Aliases:            []string{"engineering@example.com"},
		NonEditableAliases: []string{"eng@alias.example.com"},
		Etag:               "etag-1",
		Kind:               "admin#directory#group",
	}

	info := groupInfo(group)
	if info.ID != "group-id" {
		t.Fatalf("ID = %q", info.ID)
	}
	if info.DirectMembersCount != 12 {
		t.Fatalf("DirectMembersCount = %d", info.DirectMembersCount)
	}
	if info.Etag != "etag-1" {
		t.Fatalf("Etag = %q", info.Etag)
	}
}

func TestMemberInfoMapsDirectoryMemberFields(t *testing.T) {
	member := &admin.Member{
		Email:            "ada@example.com",
		Id:               "member-id",
		Role:             "OWNER",
		Type:             "USER",
		Status:           "ACTIVE",
		DeliverySettings: "ALL_MAIL",
		Etag:             "etag-1",
		Kind:             "admin#directory#member",
	}

	info := memberInfo(member)
	if info.ID != "member-id" {
		t.Fatalf("ID = %q", info.ID)
	}
	if info.DeliverySettings != "ALL_MAIL" {
		t.Fatalf("DeliverySettings = %q", info.DeliverySettings)
	}
	if info.Kind != "admin#directory#member" {
		t.Fatalf("Kind = %q", info.Kind)
	}
}

func TestCollectUserPagesFetchAll(t *testing.T) {
	calls := 0
	users, err := collectUserPages(UserListOptions{FetchAll: true}, func(pageToken string, pageSize int64) (*admin.Users, error) {
		calls++
		switch pageToken {
		case "":
			if pageSize != 500 {
				t.Fatalf("first pageSize = %d", pageSize)
			}
			return &admin.Users{
				Users: []*admin.User{
					{PrimaryEmail: "ada@example.com"},
					{PrimaryEmail: "grace@example.com"},
				},
				NextPageToken: "page-2",
			}, nil
		case "page-2":
			if pageSize != 500 {
				t.Fatalf("second pageSize = %d", pageSize)
			}
			return &admin.Users{
				Users: []*admin.User{
					{PrimaryEmail: "linus@example.com"},
				},
			}, nil
		default:
			return nil, fmt.Errorf("unexpected page token %q", pageToken)
		}
	})
	if err != nil {
		t.Fatalf("collectUserPages() error = %v", err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d", calls)
	}
	if len(users) != 3 {
		t.Fatalf("len(users) = %d", len(users))
	}
	if users[2].PrimaryEmail != "linus@example.com" {
		t.Fatalf("last user = %#v", users[2])
	}
}

func TestCollectUserPagesHonorsLimitAcrossPages(t *testing.T) {
	pageSizes := []int64{}
	users, err := collectUserPages(UserListOptions{Limit: 3}, func(pageToken string, pageSize int64) (*admin.Users, error) {
		pageSizes = append(pageSizes, pageSize)
		switch pageToken {
		case "":
			return &admin.Users{
				Users: []*admin.User{
					{PrimaryEmail: "u1@example.com"},
					{PrimaryEmail: "u2@example.com"},
				},
				NextPageToken: "page-2",
			}, nil
		case "page-2":
			if pageSize != 1 {
				t.Fatalf("second pageSize = %d, want 1", pageSize)
			}
			return &admin.Users{
				Users: []*admin.User{
					{PrimaryEmail: "u3@example.com"},
					{PrimaryEmail: "u4@example.com"},
				},
				NextPageToken: "page-3",
			}, nil
		default:
			return nil, fmt.Errorf("unexpected page token %q", pageToken)
		}
	})
	if err != nil {
		t.Fatalf("collectUserPages() error = %v", err)
	}
	if len(users) != 3 {
		t.Fatalf("len(users) = %d", len(users))
	}
	if len(pageSizes) != 2 || pageSizes[0] != 3 || pageSizes[1] != 1 {
		t.Fatalf("pageSizes = %#v", pageSizes)
	}
}

func TestCollectGroupPagesFetchAll(t *testing.T) {
	calls := 0
	groups, err := collectGroupPages(GroupListOptions{FetchAll: true}, func(pageToken string, pageSize int64) (*admin.Groups, error) {
		calls++
		switch pageToken {
		case "":
			if pageSize != 200 {
				t.Fatalf("first pageSize = %d", pageSize)
			}
			return &admin.Groups{
				Groups: []*admin.Group{
					{Email: "eng@example.com"},
				},
				NextPageToken: "page-2",
			}, nil
		case "page-2":
			if pageSize != 200 {
				t.Fatalf("second pageSize = %d", pageSize)
			}
			return &admin.Groups{
				Groups: []*admin.Group{
					{Email: "sales@example.com"},
				},
			}, nil
		default:
			return nil, fmt.Errorf("unexpected page token %q", pageToken)
		}
	})
	if err != nil {
		t.Fatalf("collectGroupPages() error = %v", err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d", calls)
	}
	if len(groups) != 2 {
		t.Fatalf("len(groups) = %d", len(groups))
	}
}

func TestCollectGroupPagesHonorsLimitAcrossPages(t *testing.T) {
	pageSizes := []int64{}
	groups, err := collectGroupPages(GroupListOptions{Limit: 2}, func(pageToken string, pageSize int64) (*admin.Groups, error) {
		pageSizes = append(pageSizes, pageSize)
		switch pageToken {
		case "":
			return &admin.Groups{
				Groups: []*admin.Group{
					{Email: "g1@example.com"},
				},
				NextPageToken: "page-2",
			}, nil
		case "page-2":
			if pageSize != 1 {
				t.Fatalf("second pageSize = %d, want 1", pageSize)
			}
			return &admin.Groups{
				Groups: []*admin.Group{
					{Email: "g2@example.com"},
					{Email: "g3@example.com"},
				},
				NextPageToken: "page-3",
			}, nil
		default:
			return nil, fmt.Errorf("unexpected page token %q", pageToken)
		}
	})
	if err != nil {
		t.Fatalf("collectGroupPages() error = %v", err)
	}
	if len(groups) != 2 {
		t.Fatalf("len(groups) = %d", len(groups))
	}
	if len(pageSizes) != 2 || pageSizes[0] != 2 || pageSizes[1] != 1 {
		t.Fatalf("pageSizes = %#v", pageSizes)
	}
}

func TestCollectMemberPagesFetchAll(t *testing.T) {
	calls := 0
	members, err := collectMemberPages(0, true, func(pageToken string, pageSize int64) (*admin.Members, error) {
		calls++
		switch pageToken {
		case "":
			if pageSize != 200 {
				t.Fatalf("first pageSize = %d", pageSize)
			}
			return &admin.Members{
				Members: []*admin.Member{
					{Email: "ada@example.com"},
				},
				NextPageToken: "page-2",
			}, nil
		case "page-2":
			if pageSize != 200 {
				t.Fatalf("second pageSize = %d", pageSize)
			}
			return &admin.Members{
				Members: []*admin.Member{
					{Email: "grace@example.com"},
				},
			}, nil
		default:
			return nil, fmt.Errorf("unexpected page token %q", pageToken)
		}
	})
	if err != nil {
		t.Fatalf("collectMemberPages() error = %v", err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d", calls)
	}
	if len(members) != 2 {
		t.Fatalf("len(members) = %d", len(members))
	}
}

func TestCollectMemberPagesHonorsLimitAcrossPages(t *testing.T) {
	pageSizes := []int64{}
	members, err := collectMemberPages(3, false, func(pageToken string, pageSize int64) (*admin.Members, error) {
		pageSizes = append(pageSizes, pageSize)
		switch pageToken {
		case "":
			return &admin.Members{
				Members: []*admin.Member{
					{Email: "m1@example.com"},
					{Email: "m2@example.com"},
				},
				NextPageToken: "page-2",
			}, nil
		case "page-2":
			if pageSize != 1 {
				t.Fatalf("second pageSize = %d, want 1", pageSize)
			}
			return &admin.Members{
				Members: []*admin.Member{
					{Email: "m3@example.com"},
					{Email: "m4@example.com"},
				},
				NextPageToken: "page-3",
			}, nil
		default:
			return nil, fmt.Errorf("unexpected page token %q", pageToken)
		}
	})
	if err != nil {
		t.Fatalf("collectMemberPages() error = %v", err)
	}
	if len(members) != 3 {
		t.Fatalf("len(members) = %d", len(members))
	}
	if len(pageSizes) != 2 || pageSizes[0] != 3 || pageSizes[1] != 1 {
		t.Fatalf("pageSizes = %#v", pageSizes)
	}
}

func TestOrgUnitInfoMapsDirectoryOrgUnitFields(t *testing.T) {
	ou := &admin.OrgUnit{
		Name:              "Engineering",
		OrgUnitId:         "ou-id",
		OrgUnitPath:       "/Engineering",
		ParentOrgUnitId:   "parent-id",
		ParentOrgUnitPath: "/",
		Description:       "Engineering team",
		BlockInheritance:  true,
		Etag:              "etag-1",
		Kind:              "admin#directory#orgUnit",
	}

	info := orgUnitInfo(ou)
	if info.OrgUnitID != "ou-id" {
		t.Fatalf("OrgUnitID = %q", info.OrgUnitID)
	}
	if !info.BlockInheritance {
		t.Fatal("BlockInheritance = false")
	}
	if info.Kind != "admin#directory#orgUnit" {
		t.Fatalf("Kind = %q", info.Kind)
	}
}
