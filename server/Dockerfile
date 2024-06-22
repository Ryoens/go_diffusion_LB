# ベースイメージとビルダーステージを指定
FROM golang:latest AS builder
 
# ワークディレクトリを設定
WORKDIR /go/myapp
 
# ソースコードをコピー
COPY integrate_multi.go /go/myapp
COPY cpu.sh /go/myapp
 
#RUN go build -o agent agent.go
#RUN go build -o integrate integrate.go
 
# 軽量化
FROM alpine:latest
WORKDIR /go/myapp
 
# bash & procps(vmstat)をインストール
RUN apk update && \
    apk add --no-cache bash procps go nano
 
COPY --from=builder /go/myapp/integrate_multi.go /go/myapp/integrate_multi.go
COPY --from=builder /go/myapp/cpu.sh /go/myapp/cpu.sh

RUN go build -o integrate integrate_multi.go 
 
# apk add bash
CMD ["/bin/bash"]