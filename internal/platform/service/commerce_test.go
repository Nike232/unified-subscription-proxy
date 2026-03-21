package service

import (
	"path/filepath"
	"testing"

	"unifiedsubscriptionproxy/internal/platform/store"
)

func TestAuthenticateUserCreatesSession(t *testing.T) {
	svc := New(store.NewFileStore(filepath.Join(t.TempDir(), "platform.json")))
	user, session, err := svc.AuthenticateUser("admin@example.com", "admin123")
	if err != nil {
		t.Fatalf("AuthenticateUser returned error: %v", err)
	}
	if user.Role != "admin" || session.Token == "" {
		t.Fatalf("unexpected auth result: %#v %#v", user, session)
	}
}

func TestCreateCheckoutOrderAndCompletePaymentCreatesSubscriptionAndKey(t *testing.T) {
	svc := New(store.NewFileStore(filepath.Join(t.TempDir(), "platform.json")))
	checkout, err := svc.CreateCheckoutOrder("user-demo", "pkg-basic", "", true, false, "http://127.0.0.1:8080")
	if err != nil {
		t.Fatalf("CreateCheckoutOrder returned error: %v", err)
	}
	if checkout.Payment.CheckoutURL == "" {
		t.Fatalf("expected checkout url")
	}

	completed, err := svc.CompletePayment(checkout.Payment.ID, "mock-ref")
	if err != nil {
		t.Fatalf("CompletePayment returned error: %v", err)
	}
	if completed.Order.Status != "paid" || completed.Payment.Status != "paid" {
		t.Fatalf("expected paid result, got %#v %#v", completed.Order, completed.Payment)
	}
	if completed.APIKey == nil || completed.APIKey.PackageID != "pkg-basic" {
		t.Fatalf("expected api key to be created: %#v", completed.APIKey)
	}
}

func TestUserPackagesMarksSubscribedPackage(t *testing.T) {
	svc := New(store.NewFileStore(filepath.Join(t.TempDir(), "platform.json")))
	packages, err := svc.UserPackages("user-demo")
	if err != nil {
		t.Fatalf("UserPackages returned error: %v", err)
	}
	if len(packages) == 0 {
		t.Fatalf("expected packages")
	}
	found := false
	for _, pkg := range packages {
		if pkg.ID == "pkg-hybrid" {
			found = true
			if !pkg.IsSubscribed {
				t.Fatalf("expected pkg-hybrid to be marked subscribed")
			}
		}
	}
	if !found {
		t.Fatalf("expected pkg-hybrid package in catalog")
	}
}

func TestConfirmUserOrderPaymentReturnsPaidDetail(t *testing.T) {
	svc := New(store.NewFileStore(filepath.Join(t.TempDir(), "platform.json")))
	checkout, err := svc.CreateCheckoutOrder("user-demo", "pkg-basic", "", true, false, "http://127.0.0.1:8080")
	if err != nil {
		t.Fatalf("CreateCheckoutOrder returned error: %v", err)
	}
	completed, err := svc.ConfirmUserOrderPayment("user-demo", checkout.Order.ID)
	if err != nil {
		t.Fatalf("ConfirmUserOrderPayment returned error: %v", err)
	}
	if completed.Order.Status != "paid" || completed.Payment.Status != "paid" {
		t.Fatalf("expected paid order/payment, got %#v %#v", completed.Order, completed.Payment)
	}
	detail, err := svc.UserOrderDetail("user-demo", checkout.Order.ID)
	if err != nil {
		t.Fatalf("UserOrderDetail returned error: %v", err)
	}
	if detail.Subscription == nil || detail.Subscription.PackageID != "pkg-basic" {
		t.Fatalf("expected order detail subscription for pkg-basic: %#v", detail.Subscription)
	}
}

func TestValidateAPIKeyRejectsExpiredSubscription(t *testing.T) {
	data := store.BootstrapData()
	data.Subscriptions[0].ExpiresAt = data.Subscriptions[0].StartsAt
	if _, _, _, err := ValidateAPIKeyInData(data, "usp_demo_key"); err == nil {
		t.Fatalf("expected expired subscription to reject api key")
	}
}

func TestRevokeUserAPIKey(t *testing.T) {
	svc := New(store.NewFileStore(filepath.Join(t.TempDir(), "platform.json")))
	key, err := svc.RevokeUserAPIKey("user-demo", "key-demo")
	if err != nil {
		t.Fatalf("RevokeUserAPIKey returned error: %v", err)
	}
	if key.Status != "revoked" {
		t.Fatalf("expected revoked key, got %#v", key)
	}
}
