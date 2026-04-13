FROM mcr.microsoft.com/vscode/devcontainers/go
RUN sudo apt update

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

ADD https://github.com/protocolbuffers/protobuf/releases/download/v34.1/protoc-34.1-linux-x86_64.zip protoc.zip
RUN unzip protoc.zip -d /usr
RUN sudo chown -R vscode:golang /go