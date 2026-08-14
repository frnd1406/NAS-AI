FROM node:20-alpine3.21 AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:1.27-alpine3.21
# FIX: Upgrade libxml2 to 2.13.9-r0 to patch CVE-2025-49794, CVE-2025-49796 (CRITICAL)
# and CVE-2025-32414, CVE-2025-32415, CVE-2025-49795, CVE-2025-6021 (HIGH)
RUN apk upgrade --no-cache libxml2
COPY default.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
