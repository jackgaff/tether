FROM oven/bun:1.3.10 AS deps

WORKDIR /workspace

COPY package.json bun.lock ./
COPY apps/web/package.json ./apps/web/package.json
COPY apps/test/package.json ./apps/test/package.json
RUN bun install --frozen-lockfile

FROM deps AS dev

WORKDIR /workspace
EXPOSE 5173

CMD ["sh", "-lc", "cd apps/web && bun run dev --host 0.0.0.0 --port 5173"]

FROM deps AS build

WORKDIR /workspace

COPY package.json bun.lock ./
COPY apps/web /workspace/apps/web
COPY apps/test/package.json /workspace/apps/test/package.json

RUN cd apps/web && bun run build

FROM nginx:1.27-alpine AS runtime

COPY apps/web/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /workspace/apps/web/dist /usr/share/nginx/html

EXPOSE 8080
