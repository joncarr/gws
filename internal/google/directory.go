package google

import (
	"context"
	"fmt"

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

type DirectoryClient interface {
	DomainInfo(ctx context.Context, profile config.Profile) (DomainInfo, error)
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
