package service

import (
	"errors"
	"strings"
	"time"

	"unifiedsubscriptionproxy/internal/platform/domain"
)

const (
	sessionTTL             = 7 * 24 * time.Hour
	defaultPaymentProvider = "mockpay"
	defaultCurrency        = "CNY"
)

type UserProfile struct {
	User          domain.User           `json:"user"`
	Subscriptions []domain.Subscription `json:"subscriptions"`
	APIKeys       []domain.APIKey       `json:"api_keys"`
	Orders        []domain.Order        `json:"orders"`
	Payments      []domain.Payment      `json:"payments"`
}

type CheckoutResult struct {
	Order   domain.Order   `json:"order"`
	Payment domain.Payment `json:"payment"`
	APIKey  *domain.APIKey `json:"api_key,omitempty"`
}

func (s *Service) AuthenticateUser(email, password string) (domain.User, domain.AuthSession, error) {
	var user domain.User
	var session domain.AuthSession
	_, err := s.store.Mutate(func(data *domain.PlatformData) error {
		for _, candidate := range data.Users {
			if !strings.EqualFold(strings.TrimSpace(candidate.Email), strings.TrimSpace(email)) {
				continue
			}
			if candidate.PasswordHash != password {
				return errors.New("invalid email or password")
			}
			user = candidate
			session = domain.AuthSession{
				ID:        "sess-" + randomID(6),
				Token:     "usp_sess_" + randomID(16),
				UserID:    candidate.ID,
				Role:      candidate.Role,
				CreatedAt: time.Now().UTC(),
				ExpiresAt: time.Now().UTC().Add(sessionTTL),
			}
			data.AuthSessions = append(data.AuthSessions, session)
			return nil
		}
		return errors.New("invalid email or password")
	})
	return user, session, err
}

func (s *Service) SessionUser(token string) (domain.User, domain.AuthSession, error) {
	data, err := s.store.Load()
	if err != nil {
		return domain.User{}, domain.AuthSession{}, err
	}
	now := time.Now().UTC()
	for _, session := range data.AuthSessions {
		if session.Token != token || !session.ExpiresAt.After(now) {
			continue
		}
		for _, user := range data.Users {
			if user.ID == session.UserID {
				return user, session, nil
			}
		}
	}
	return domain.User{}, domain.AuthSession{}, errors.New("session not found")
}

func (s *Service) RevokeSession(token string) error {
	_, err := s.store.Mutate(func(data *domain.PlatformData) error {
		out := make([]domain.AuthSession, 0, len(data.AuthSessions))
		for _, session := range data.AuthSessions {
			if session.Token == token {
				continue
			}
			out = append(out, session)
		}
		data.AuthSessions = out
		return nil
	})
	return err
}

func (s *Service) CleanupAuthSessions(now time.Time) (int, error) {
	removed := 0
	_, err := s.store.Mutate(func(data *domain.PlatformData) error {
		out := make([]domain.AuthSession, 0, len(data.AuthSessions))
		for _, session := range data.AuthSessions {
			if !session.ExpiresAt.After(now) {
				removed++
				continue
			}
			out = append(out, session)
		}
		data.AuthSessions = out
		return nil
	})
	return removed, err
}

func (s *Service) UserProfile(userID string) (UserProfile, error) {
	data, err := s.store.Load()
	if err != nil {
		return UserProfile{}, err
	}
	return userProfileFromData(data, userID)
}

func userProfileFromData(data domain.PlatformData, userID string) (UserProfile, error) {
	var profile UserProfile
	for _, user := range data.Users {
		if user.ID == userID {
			profile.User = user
			break
		}
	}
	if profile.User.ID == "" {
		return UserProfile{}, errors.New("user not found")
	}
	now := time.Now().UTC()
	for _, sub := range data.Subscriptions {
		if sub.UserID != userID {
			continue
		}
		if sub.Status == domain.SubscriptionStatusActive && !sub.ExpiresAt.After(now) {
			sub.Status = domain.SubscriptionStatusExpired
		}
		profile.Subscriptions = append(profile.Subscriptions, sub)
	}
	for _, key := range data.APIKeys {
		if key.UserID == userID {
			profile.APIKeys = append(profile.APIKeys, key)
		}
	}
	for _, order := range data.Orders {
		if order.UserID == userID {
			profile.Orders = append(profile.Orders, order)
		}
	}
	for _, payment := range data.Payments {
		if payment.UserID == userID {
			profile.Payments = append(profile.Payments, payment)
		}
	}
	return profile, nil
}

func (s *Service) UserUsageLogs(userID string) ([]domain.UsageLog, error) {
	data, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	out := make([]domain.UsageLog, 0)
	for i := len(data.UsageLogs) - 1; i >= 0; i-- {
		if data.UsageLogs[i].UserID == userID {
			out = append(out, data.UsageLogs[i])
		}
	}
	return out, nil
}

func (s *Service) UserPackages() ([]domain.ServicePackage, error) {
	data, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	out := make([]domain.ServicePackage, 0)
	for _, pkg := range data.ServicePackages {
		if pkg.IsActive {
			out = append(out, pkg)
		}
	}
	return out, nil
}

func (s *Service) CreateCheckoutOrder(userID, packageID, bindAPIKeyID string, createAPIKey, autoRenew bool, checkoutBaseURL string) (CheckoutResult, error) {
	var result CheckoutResult
	_, err := s.store.Mutate(func(data *domain.PlatformData) error {
		var pkg domain.ServicePackage
		foundPackage := false
		for _, candidate := range data.ServicePackages {
			if candidate.ID == packageID && candidate.IsActive {
				pkg = candidate
				foundPackage = true
				break
			}
		}
		if !foundPackage {
			return errors.New("package not found")
		}
		if bindAPIKeyID != "" {
			owned := false
			for _, key := range data.APIKeys {
				if key.ID == bindAPIKeyID && key.UserID == userID {
					owned = true
					break
				}
			}
			if !owned {
				return errors.New("api key not found for user")
			}
		}

		order := domain.Order{
			ID:           "order-" + randomID(6),
			UserID:       userID,
			PackageID:    packageID,
			Status:       "pending",
			AmountCents:  pkg.PriceCents,
			Currency:     defaultCurrency,
			BillingCycle: defaultBillingCycle(pkg.BillingCycle),
			AutoRenew:    autoRenew,
			BindAPIKeyID: bindAPIKeyID,
			CreateAPIKey: createAPIKey,
			CreatedAt:    time.Now().UTC(),
		}
		payment := domain.Payment{
			ID:          "pay-" + randomID(6),
			OrderID:     order.ID,
			UserID:      userID,
			Provider:    defaultPaymentProvider,
			Status:      "pending",
			AmountCents: order.AmountCents,
			Currency:    order.Currency,
			CreatedAt:   time.Now().UTC(),
		}
		payment.CheckoutURL = strings.TrimRight(checkoutBaseURL, "/") + "/mockpay/checkout?payment_id=" + payment.ID
		order.PaymentID = payment.ID

		data.Orders = append(data.Orders, order)
		data.Payments = append(data.Payments, payment)

		result.Order = order
		result.Payment = payment
		return nil
	})
	return result, err
}

func (s *Service) CompletePayment(paymentID, providerRef string) (CheckoutResult, error) {
	var result CheckoutResult
	_, err := s.store.Mutate(func(data *domain.PlatformData) error {
		paymentIndex := -1
		for i := range data.Payments {
			if data.Payments[i].ID == paymentID {
				paymentIndex = i
				break
			}
		}
		if paymentIndex == -1 {
			return errors.New("payment not found")
		}
		payment := &data.Payments[paymentIndex]
		now := time.Now().UTC()
		payment.Status = "paid"
		payment.ProviderRef = strings.TrimSpace(providerRef)
		payment.CompletedAt = now
		payment.WebhookStatus = "processed"

		event := domain.WebhookEvent{
			ID:         "wh-" + randomID(6),
			Provider:   payment.Provider,
			EventType:  "payment.paid",
			PaymentID:  payment.ID,
			Status:     payment.Status,
			ReceivedAt: now,
		}
		data.WebhookEvents = append(data.WebhookEvents, event)

		orderIndex := -1
		for i := range data.Orders {
			if data.Orders[i].ID == payment.OrderID {
				orderIndex = i
				break
			}
		}
		if orderIndex == -1 {
			return errors.New("order not found")
		}
		order := &data.Orders[orderIndex]
		order.Status = "paid"
		order.CompletedAt = now

		startAt := now
		subscriptionIndex := -1
		for i := range data.Subscriptions {
			if data.Subscriptions[i].UserID == order.UserID && data.Subscriptions[i].PackageID == order.PackageID {
				subscriptionIndex = i
				break
			}
		}
		if subscriptionIndex >= 0 {
			if data.Subscriptions[subscriptionIndex].ExpiresAt.After(now) {
				startAt = data.Subscriptions[subscriptionIndex].ExpiresAt
			}
			data.Subscriptions[subscriptionIndex].Status = domain.SubscriptionStatusActive
			data.Subscriptions[subscriptionIndex].OrderID = order.ID
			data.Subscriptions[subscriptionIndex].AutoRenew = order.AutoRenew
			data.Subscriptions[subscriptionIndex].ExpiresAt = startAt.Add(billingCycleDuration(order.BillingCycle))
			result.Order.SubscriptionID = data.Subscriptions[subscriptionIndex].ID
			order.SubscriptionID = data.Subscriptions[subscriptionIndex].ID
		} else {
			sub := domain.Subscription{
				ID:          "sub-" + randomID(6),
				UserID:      order.UserID,
				PackageID:   order.PackageID,
				OrderID:     order.ID,
				Status:      domain.SubscriptionStatusActive,
				StartsAt:    now,
				ExpiresAt:   startAt.Add(billingCycleDuration(order.BillingCycle)),
				AssignedBy:  order.UserID,
				Description: "self-service purchase",
				AutoRenew:   order.AutoRenew,
			}
			data.Subscriptions = append(data.Subscriptions, sub)
			order.SubscriptionID = sub.ID
		}

		var createdKey *domain.APIKey
		if order.BindAPIKeyID != "" {
			for i := range data.APIKeys {
				if data.APIKeys[i].ID == order.BindAPIKeyID {
					data.APIKeys[i].PackageID = order.PackageID
					break
				}
			}
		} else if order.CreateAPIKey {
			key := domain.APIKey{
				ID:        "key-" + randomID(4),
				Key:       "usp_" + randomID(12),
				UserID:    order.UserID,
				PackageID: order.PackageID,
				Status:    "active",
				CreatedAt: now,
			}
			data.APIKeys = append(data.APIKeys, key)
			createdKey = &key
		}

		result.Order = *order
		result.Payment = *payment
		result.APIKey = createdKey
		return nil
	})
	return result, err
}

func (s *Service) CreateUserAPIKey(userID, packageID string) (domain.APIKey, error) {
	var key domain.APIKey
	_, err := s.store.Mutate(func(data *domain.PlatformData) error {
		if !hasActiveSubscription(*data, userID, packageID) {
			return errors.New("active subscription required for package")
		}
		key = domain.APIKey{
			ID:        "key-" + randomID(4),
			Key:       "usp_" + randomID(12),
			UserID:    userID,
			PackageID: packageID,
			Status:    "active",
			CreatedAt: time.Now().UTC(),
		}
		data.APIKeys = append(data.APIKeys, key)
		return nil
	})
	return key, err
}

func (s *Service) RevokeUserAPIKey(userID, keyID string) (domain.APIKey, error) {
	var key domain.APIKey
	_, err := s.store.Mutate(func(data *domain.PlatformData) error {
		for i := range data.APIKeys {
			if data.APIKeys[i].ID != keyID || data.APIKeys[i].UserID != userID {
				continue
			}
			data.APIKeys[i].Status = "revoked"
			key = data.APIKeys[i]
			return nil
		}
		return errors.New("api key not found")
	})
	return key, err
}

func hasActiveSubscription(data domain.PlatformData, userID, packageID string) bool {
	now := time.Now().UTC()
	for _, sub := range data.Subscriptions {
		if sub.UserID == userID && sub.PackageID == packageID && sub.Status == domain.SubscriptionStatusActive && sub.ExpiresAt.After(now) {
			return true
		}
	}
	return false
}

func defaultBillingCycle(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "monthly"
	}
	return value
}

func billingCycleDuration(value string) time.Duration {
	switch strings.TrimSpace(value) {
	case "yearly":
		return 365 * 24 * time.Hour
	default:
		return 30 * 24 * time.Hour
	}
}
