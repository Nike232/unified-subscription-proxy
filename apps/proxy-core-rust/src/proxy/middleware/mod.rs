// Middleware 模块 - Axum 中间件

pub mod auth;
pub mod cors;
pub mod ip_filter;
pub mod logging;
pub mod monitor;
pub mod service_status;
pub mod rate_limit;

pub use auth::{admin_auth_middleware, auth_middleware};
pub use cors::cors_layer;
pub use ip_filter::ip_filter_middleware;
pub use monitor::{monitor_middleware, ProxyRequestLog};
pub use service_status::service_status_middleware;
pub use rate_limit::rate_limit_middleware;
pub use auth::{auth_middleware, admin_auth_middleware};
pub use ip_filter::ip_filter_middleware;
