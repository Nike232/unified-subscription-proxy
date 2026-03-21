FROM rust:1.85-slim AS builder

RUN apt-get update && apt-get install -y pkg-config libssl-dev build-essential sqlite3 libsqlite3-dev \
    libgtk-3-dev libsoup-3.0-dev libjavascriptcoregtk-4.1-dev libwebkit2gtk-4.1-dev \
    libayatana-appindicator3-dev librsvg2-dev cmake perl golang

WORKDIR /app
COPY . .

# Since this is a workspace-like structure but we are in a subfolder, 
# we need to ensure Cargo.toml is correct.
RUN cargo update -p time --precise 0.3.36
RUN cargo build --release

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y libssl3 ca-certificates sqlite3 libsqlite3-0 && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /app/target/release/antigravity_tools_lib .
# Rename if necessary or adjust entrypoint
# The binary name should be available in target/release/
CMD ["./antigravity_tools_lib", "--headless"]
