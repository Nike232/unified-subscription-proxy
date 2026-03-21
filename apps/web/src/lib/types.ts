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
