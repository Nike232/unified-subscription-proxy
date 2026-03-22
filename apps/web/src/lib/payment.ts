export const paymentQrImageUrl =
  (import.meta.env.VITE_PAYMENT_QR_IMAGE_URL as string | undefined)?.trim() || "/payment-qr.svg";

export const paymentQrNote =
  (import.meta.env.VITE_PAYMENT_QR_NOTE as string | undefined)?.trim() ||
  "请使用管理员提供的测试收款方式扫码付款。付款完成后返回订单页，点击“我已付款，提交确认”。";
