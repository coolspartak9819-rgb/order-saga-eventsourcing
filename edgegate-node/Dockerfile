FROM node:22-alpine
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --omit=dev
COPY src ./src
COPY plugins ./plugins
COPY config.example.json ./config.json
USER node
EXPOSE 8080
ENV CONFIG_PATH=/app/config.json
CMD ["node", "src/index.js"]
