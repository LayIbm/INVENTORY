FROM node:20-alpine

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY package*.json ./
RUN npm install

COPY . .

RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 5173
CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0"]
