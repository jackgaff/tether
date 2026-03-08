FROM oven/bun:1.3.10

WORKDIR /workspace

COPY package.json bun.lock ./
COPY apps/web/package.json ./apps/web/package.json
COPY apps/test/package.json ./apps/test/package.json
RUN bun install --frozen-lockfile

EXPOSE 5174

CMD ["sh", "-lc", "cd apps/test && bun run dev --host 0.0.0.0 --port 5174"]
