@echo off
cd /d "%~dp0"
go test ./...
pause
