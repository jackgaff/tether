FROM oven/bun:1.3.10

WORKDIR /workspace

COPY package.json bun.lock ./
COPY apps/web/package.json ./apps/web/package.json
RUN bun install --frozen-lockfile

EXPOSE 5173

CMD ["sh", "-lc", "cd apps/web && bun run dev --host 0.0.0.0 --port 5173"]
