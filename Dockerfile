# build stage
FROM golang:alpine AS build
WORKDIR /build
COPY . .
RUN go build -buildvcs=false -o app -v

# final stage
FROM alpine:latest
WORKDIR /dist
RUN mkdir views
RUN mkdir -p public/css
RUN mkdir -p public/js
COPY --from=build /build/app .
COPY --from=build /build/views/*.gohtml views/
COPY --from=build /build/public/css/*.css public/css/
COPY --from=build /build/public/js/*.js public/js/
EXPOSE 4000
CMD ./app -bind :4000
