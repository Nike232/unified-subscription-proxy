FROM rust:1.85-slim AS builder

RUN apt-get update && apt-get install -y pkg-config libssl-dev build-essential sqlite3 libsqlite3-dev \
    libgtk-3-dev libsoup-3.0-dev libjavascriptcoregtk-4.1-dev libwebkit2gtk-4.1-dev \
    libayatana-appindicator3-dev librsvg2-dev cmake perl golang

WORKDIR /app
COPY . .

RUN cargo build --release --locked --bin antigravity_tools

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y curl libssl3 ca-certificates sqlite3 libsqlite3-0 && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /app/target/release/antigravity_tools .
EXPOSE 8045
CMD ["./antigravity_tools", "--headless"]
