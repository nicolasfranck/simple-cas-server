# build stage
FROM golang:alpine AS build
WORKDIR /build
COPY . .
RUN go build -buildvcs=false -o app -v

# final stage
FROM alpine:latest
WORKDIR /dist
COPY --from=build /build/app .
COPY --from=build /build/views .
RUN mkdir views && mv *.gohtml views
EXPOSE 4000
CMD ls -alhtr && ./app -bind :4000
