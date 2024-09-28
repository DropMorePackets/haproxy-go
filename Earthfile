VERSION 0.8

deps:
    FROM golang:1.23.1-alpine
    ENV CGO_ENABLED 0
    WORKDIR /src
    COPY . .
    RUN go work init
    RUN go work use .
    RUN go work use ./internal/tools
    RUN go work use ./peers
    RUN go work use ./spop
    RUN go mod download

tools:
    FROM +deps

    RUN go install honnef.co/go/tools/cmd/staticcheck
    RUN go install golang.org/x/tools/cmd/stringer
    RUN go install ./internal/tools/vet

validate:
    FROM +tools
    RUN $GOPATH/bin/staticcheck ./...
    RUN $GOPATH/bin/vet ./...

go-test:
    FROM +deps
    RUN mkdir e2e
    FOR target IN './spop' './peers' './...'
        RUN go test $target
        RUN go test -tags e2e -o e2e -c $target
    END
    SAVE ARTIFACT e2e

e2e:
    FROM haproxy:2.9-alpine
    COPY +go-test/e2e e2e
    FOR test IN $(ls ./e2e)
        RUN ./e2e/$test
    END

test:
    BUILD +validate
    BUILD +e2e