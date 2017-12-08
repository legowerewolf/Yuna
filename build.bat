@ECHO off

ECHO [Go] Setting variables...
SET CGO_ENABLED=0
SET GOOS=linux

ECHO [Go] Building Yuna executable...
go build -a -installsuffix cgo -o yuna .

ECHO [Docker] Setting variables...
@FOR /f "tokens=*" %%i IN ('docker-machine env default') DO @%%i

ECHO [Docker] Packaging...
docker build . -t legowerewolf/yuna:latest

ECHO [Cleanup]
DEL yuna