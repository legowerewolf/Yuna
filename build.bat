@ECHO off

ECHO [Docker] Setting variables...
@FOR /f "tokens=*" %%i IN ('docker-machine env default') DO @%%i

ECHO [Docker] Packaging...
docker build . -t legowerewolf/yuna:latest