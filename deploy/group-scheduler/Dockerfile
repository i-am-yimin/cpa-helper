FROM golang:1.26-alpine AS build
RUN apk add --no-cache gcc musl-dev
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
COPY backend/cmd/cpa-group-scheduler ./cmd/cpa-group-scheduler
COPY backend/internal/groupscheduler ./internal/groupscheduler
RUN CGO_ENABLED=1 go build -buildmode=c-shared -o /out/cpa-group-scheduler.so ./cmd/cpa-group-scheduler

FROM scratch
COPY --from=build /out/cpa-group-scheduler.so /
COPY --from=build /out/cpa-group-scheduler.h /
