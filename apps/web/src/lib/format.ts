export function formatCurrency(cents?: number, currency = "CNY") {
  return new Intl.NumberFormat("zh-CN", {
    style: "currency",
    currency,
    minimumFractionDigits: 2,
  }).format((cents ?? 0) / 100);
}

export function formatDate(value?: string) {
  if (!value || value.startsWith("0001-01-01")) {
    return "暂无记录";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "暂无记录";
  }
  return date.toLocaleString("zh-CN");
}

export function mapSubscriptionStatus(status?: string) {
  switch (status) {
    case "active":
      return "生效中";
    case "expired":
      return "已过期";
    default:
      return status || "未知";
  }
}

export function mapPaymentStatus(status?: string) {
  switch (status) {
    case "pending":
      return "待付款";
    case "paid":
      return "已付款";
    default:
      return status || "未知";
  }
}

export function mapKeyStatus(status?: string) {
  switch (status) {
    case "active":
      return "可用";
    case "revoked":
      return "已吊销";
    default:
      return status || "未知";
  }
}

export function mapOrderStatus(status?: string) {
  switch (status) {
    case "pending":
      return "待处理";
    case "paid":
      return "已支付";
    default:
      return status || "未知";
  }
}

export function mapUsageStatus(status?: string) {
  switch (status) {
    case "completed":
      return "成功";
    case "failed":
      return "失败";
    default:
      return status || "未知";
  }
}
