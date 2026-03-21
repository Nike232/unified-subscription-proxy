use axum::{
    extract::{Request, State},
    http::StatusCode,
    middleware::Next,
};
use dashmap::DashMap;
use std::sync::Arc;
use tokio::time::Instant;

use crate::proxy::middleware::auth::UserTokenIdentity;
use crate::proxy::server::AppState;

#[derive(Clone)]
pub struct RateLimiter {
    // map of token_id -> (rpm_tokens, last_rpm_refill, tpm_tokens, last_tpm_refill)
    buckets: Arc<DashMap<String, (f64, Instant, f64, Instant)>>,
}

impl RateLimiter {
    pub fn new() -> Self {
        Self {
            buckets: Arc::new(DashMap::new()),
        }
    }

    pub fn check_limit(&self, token_id: &str, max_rpm: i64, max_tpm: i64, estimated_tokens: i64) -> bool {
        if max_rpm <= 0 && max_tpm <= 0 {
            return true;
        }

        let now = Instant::now();
        let mut entry = self.buckets.entry(token_id.to_string()).or_insert_with(|| {
            (max_rpm as f64, now, max_tpm as f64, now)
        });

        let (mut rpm_tokens, mut last_rpm, mut tpm_tokens, mut last_tpm) = *entry;

        if max_rpm > 0 {
            let elapsed_rpm = now.duration_since(last_rpm).as_secs_f64();
            rpm_tokens = (rpm_tokens + elapsed_rpm * (max_rpm as f64 / 60.0)).min(max_rpm as f64);
            last_rpm = now;
        }

        if max_tpm > 0 {
            let elapsed_tpm = now.duration_since(last_tpm).as_secs_f64();
            tpm_tokens = (tpm_tokens + elapsed_tpm * (max_tpm as f64 / 60.0)).min(max_tpm as f64);
            last_tpm = now;
        }

        let mut allowed = true;
        if max_rpm > 0 {
            if rpm_tokens >= 1.0 {
                rpm_tokens -= 1.0;
            } else {
                allowed = false;
            }
        }

        if allowed && max_tpm > 0 {
            if tpm_tokens >= estimated_tokens as f64 {
                tpm_tokens -= estimated_tokens as f64;
            } else {
                allowed = false;
            }
        }

        if allowed {
            *entry = (rpm_tokens, last_rpm, tpm_tokens, last_tpm);
        }

        allowed
    }
}

pub async fn rate_limit_middleware(
    State(state): State<AppState>,
    request: Request,
    next: Next,
) -> Result<axum::response::Response, StatusCode> {
    
    // Estimate tokens roughly based on payload size or a fixed batch
    // Because parsing JSON stream here is expensive, we rely mainly on PRPM
    // or fix a small base for PTPM.
    let estimated_tokens = 500; 

    if let Some(identity) = request.extensions().get::<UserTokenIdentity>() {
        // Double check zero total quota and non monthly sub bypass
        if identity.total_quota <= 0.0 && identity.group_id != "gemini_vip" {
            // Strictly they shouldn't reach here if Go backend blocks them, but double filter
            tracing::warn!("Rejecting {} due to Zero TotalQuota", identity.username);
            return Err(StatusCode::PAYMENT_REQUIRED);
        }

        if identity.rpm > 0 || identity.tpm > 0 {
            let limiter = state.rate_limiter.clone();
            if !limiter.check_limit(&identity.token_id, identity.rpm, identity.tpm, estimated_tokens) {
                tracing::warn!("Rate limit exceeded for token_id: {} (rpm: {}, tpm: {})", identity.token_id, identity.rpm, identity.tpm);
                return Err(StatusCode::TOO_MANY_REQUESTS);
            }
        }
    }

    Ok(next.run(request).await)
}
