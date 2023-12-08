
##
## STEP 1 - BUILD
##
FROM golang:1.21 AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ecs-template-server/*.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /bootstrap


##
## STEP 2 - DEPLOY
##
FROM scratch

WORKDIR /

COPY --from=build /bootstrap /bootstrap

EXPOSE 80

ENTRYPOINT ["/bootstrap"]