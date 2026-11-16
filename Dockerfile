FROM node:22 AS web
WORKDIR /web
COPY web/package*.json ./
RUN npm ci
COPY web/ .
RUN npx ng build --configuration production

FROM golang:1.27 AS api
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app ./cmd/api

FROM gcr.io/distroless/static
COPY --from=api /app /app
COPY --from=web /web/dist/web/browser /public
ENV PUBLIC_DIR=/public
EXPOSE 3000
ENTRYPOINT ["/app"]