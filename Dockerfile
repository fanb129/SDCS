FROM ubuntu:20.04
LABEL authors="fanb"

WORKDIR /root
COPY . .

# 安装protoc wget golang1.23
RUN apt update && apt install -y protobuf-compiler wget
RUN wget https://go.dev/dl/go1.23.1.linux-amd64.tar.gz
RUN rm -rf /usr/local/go && tar -C /usr/local -xzf go1.23.1.linux-amd64.tar.gz
ENV GOPATH="/root/go"
ENV GOROOT="/usr/local/go"
ENV PATH="$GOPATH/bin:$GOROOT/bin:$PATH"

# 配置go代理
RUN go env -w GO111MODULE=on \
    && go env -w GOPROXY=https://goproxy.cn,direct \
    && go env

# 安装protoc插件
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
# 编译proto文件
RUN protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/cache.proto

# 编译sdcs
RUN go mod tidy \
    && go build -o sdcs ./cmd/sdcs_server/main.go

EXPOSE 8080

CMD ["/root/sdcs"]