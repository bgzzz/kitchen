FROM golang as builder 

WORKDIR /app

COPY Makefile go.mod go.sum ./
RUN make mod tools 

COPY . .
RUN make test-all build

FROM scratch

COPY --from=builder /app/bin/kitchen /kitchen

ENTRYPOINT [ "/kitchen" ]