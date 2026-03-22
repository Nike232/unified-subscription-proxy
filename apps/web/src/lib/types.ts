export interface SessionUser {
  id: string;
  email: string;
  name: string;
  role: string;
  status?: string;
}

export interface UserCatalogItem {
  id: string;
  name: string;
  display_name?: string;
  tier?: string;
  description: string;
  price_cents: number;
  billing_cycle?: string;
  provider_access: Array<{
    provider: string;
    models: string[];
  }>;
  is_subscribed: boolean;
  user_status?: string;
}

export interface UserSubscriptionItem {
  id: string;
  package_id: string;
  order_id?: string;
  status: string;
  expires_at: string;
  starts_at: string;
  auto_renew?: boolean;
}

export interface UserAPIKeyItem {
  id: string;
  key: string;
  package_id: string;
  status: string;
  created_at: string;
  last_used_at?: string;
}

export interface UserOrderItem {
  id: string;
  package_id: string;
  status: string;
  amount_cents: number;
  currency: string;
  billing_cycle: string;
  payment_id?: string;
  subscription_id?: string;
  created_at: string;
  completed_at?: string;
  create_api_key?: boolean;
}

export interface PaymentDetail {
  id: string;
  order_id: string;
  status: string;
  amount_cents: number;
  currency: string;
  checkout_url?: string;
  provider_ref?: string;
  created_at: string;
  completed_at?: string;
}

export interface UserOrderDetail {
  order: UserOrderItem;
  payment?: PaymentDetail;
  package?: UserCatalogItem;
  subscription?: UserSubscriptionItem;
}

export interface UserUsageLogItem {
  id: string;
  model_alias: string;
  provider: string;
  status: string;
  created_at: string;
  error_type?: string;
  error_message?: string;
  upstream_model?: string;
  total_tokens?: number;
}

export interface UserProfile {
  user: SessionUser;
  subscriptions: UserSubscriptionItem[];
  api_keys: UserAPIKeyItem[];
  orders: UserOrderItem[];
  payments: PaymentDetail[];
}

export interface KernelStatus {
  mode: string;
  primary: string;
  kernels: Record<string, {
    origin: string;
    healthy: boolean;
    configured: boolean;
    role: string;
    error?: string;
  }>;
}

export interface AdminUserItem {
  id: string;
  email: string;
  name: string;
  role: string;
  status: string;
  balance: number;
  group: string;
  total_quota: number;
  rpm: number;
  tpm: number;
  concurrency: number;
}

export interface AdminUpstreamAccountItem {
  id: string;
  provider: string;
  display_name: string;
  email: string;
  auth_mode: string;
  status: string;
  tier: string;
  weight: number;
  priority: number;
  supports_models: string[];
  meta?: Record<string, string>;
  last_refreshed_at?: string;
}

export interface AdminOAuthProviderConfig {
  Provider?: string;
  ClientID?: string;
  ClientSecret?: string;
  AuthorizeURL?: string;
  TokenURL?: string;
  RedirectURL?: string;
  Scopes?: string[];
  RefreshScopes?: string[];
  Prompt?: string;
  AccessType?: string;
  UsePKCE?: boolean;
  IncludeGrantedScopes?: boolean;
  ExtraAuthorizeParams?: Record<string, string>;
}

export interface AdminPackageItem {
  id: string;
  name: string;
  display_name?: string;
  tier?: string;
  description: string;
  price_cents: number;
  billing_cycle?: string;
  is_active: boolean;
  provider_access: Array<{
    provider: string;
    models: string[];
  }>;
}

export interface AdminSubscriptionItem {
  id: string;
  user_id: string;
  package_id: string;
  order_id?: string;
  status: string;
  starts_at: string;
  expires_at: string;
  auto_renew?: boolean;
}

export interface AdminAPIKeyItem {
  id: string;
  key: string;
  user_id: string;
  package_id: string;
  status: string;
  created_at: string;
  last_used_at?: string;
}

export interface AdminUsageLogItem extends UserUsageLogItem {
  api_key_id?: string;
  user_id?: string;
  account_id?: string;
}

export interface AdminOrderItem extends UserOrderItem {
  user_id: string;
}

export interface AdminPaymentItem extends PaymentDetail {
  user_id: string;
  provider: string;
  webhook_status?: string;
}

export interface DispatchDebugResult {
  model_alias: string;
  package_id: string;
  selected_account_id: string;
  selected_provider: string;
  upstream_model: string;
  fallback_allowed: boolean;
  candidates: Array<{
    account_id: string;
    provider: string;
    upstream_model: string;
    account_status?: string;
    skip_reason?: string;
    can_refresh?: boolean;
    role?: string;
    cooldown_until?: string;
    consecutive_failures?: number;
    last_failure_reason?: string;
  }>;
}
