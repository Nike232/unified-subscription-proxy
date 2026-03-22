FROM node:22-alpine AS builder
WORKDIR /app
ARG VITE_PAYMENT_QR_IMAGE_URL=
ARG VITE_PAYMENT_QR_NOTE=
ENV VITE_PAYMENT_QR_IMAGE_URL=$VITE_PAYMENT_QR_IMAGE_URL
ENV VITE_PAYMENT_QR_NOTE=$VITE_PAYMENT_QR_NOTE
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf.template /etc/nginx/templates/default.conf.template
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
