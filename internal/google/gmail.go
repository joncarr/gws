package google

import (
	"context"
	"fmt"

	gmail "google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"

	"github.com/joncarr/gws/internal/auth"
	"github.com/joncarr/gws/internal/config"
)

type DelegateInfo struct {
	DelegateEmail      string `json:"delegate_email"`
	VerificationStatus string `json:"verification_status,omitempty"`
}

type GmailClient interface {
	Delegates(ctx context.Context, profile config.Profile, userEmail string) ([]DelegateInfo, error)
	Delegate(ctx context.Context, profile config.Profile, userEmail string, delegateEmail string) (DelegateInfo, error)
	CreateDelegate(ctx context.Context, profile config.Profile, userEmail string, delegateEmail string) (DelegateInfo, error)
	DeleteDelegate(ctx context.Context, profile config.Profile, userEmail string, delegateEmail string) error
}

type AdminGmailClient struct{}

func (AdminGmailClient) Delegates(ctx context.Context, profile config.Profile, userEmail string) ([]DelegateInfo, error) {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return nil, err
	}
	svc, err := gmail.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("create Gmail client: %w", err)
	}
	resp, err := svc.Users.Settings.Delegates.List(userEmail).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("call Gmail API: %w", err)
	}
	delegates := make([]DelegateInfo, 0, len(resp.Delegates))
	for _, delegate := range resp.Delegates {
		delegates = append(delegates, delegateInfo(delegate))
	}
	return delegates, nil
}

func (AdminGmailClient) Delegate(ctx context.Context, profile config.Profile, userEmail string, delegateEmail string) (DelegateInfo, error) {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return DelegateInfo{}, err
	}
	svc, err := gmail.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return DelegateInfo{}, fmt.Errorf("create Gmail client: %w", err)
	}
	delegate, err := svc.Users.Settings.Delegates.Get(userEmail, delegateEmail).Context(ctx).Do()
	if err != nil {
		return DelegateInfo{}, fmt.Errorf("call Gmail API: %w", err)
	}
	return delegateInfo(delegate), nil
}

func (AdminGmailClient) CreateDelegate(ctx context.Context, profile config.Profile, userEmail string, delegateEmail string) (DelegateInfo, error) {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return DelegateInfo{}, err
	}
	svc, err := gmail.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return DelegateInfo{}, fmt.Errorf("create Gmail client: %w", err)
	}
	delegate, err := svc.Users.Settings.Delegates.Create(userEmail, &gmail.Delegate{
		DelegateEmail: delegateEmail,
	}).Context(ctx).Do()
	if err != nil {
		return DelegateInfo{}, fmt.Errorf("call Gmail API: %w", err)
	}
	return delegateInfo(delegate), nil
}

func (AdminGmailClient) DeleteDelegate(ctx context.Context, profile config.Profile, userEmail string, delegateEmail string) error {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return err
	}
	svc, err := gmail.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return fmt.Errorf("create Gmail client: %w", err)
	}
	if err := svc.Users.Settings.Delegates.Delete(userEmail, delegateEmail).Context(ctx).Do(); err != nil {
		return fmt.Errorf("call Gmail API: %w", err)
	}
	return nil
}

func delegateInfo(delegate *gmail.Delegate) DelegateInfo {
	return DelegateInfo{
		DelegateEmail:      delegate.DelegateEmail,
		VerificationStatus: delegate.VerificationStatus,
	}
}
