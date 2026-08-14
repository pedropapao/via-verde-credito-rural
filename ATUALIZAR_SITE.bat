@echo off
setlocal
cd /d "%~dp0"
echo [1/3] Testando o projeto...
go test ./...
if errorlevel 1 (
  echo.
  echo A atualizacao foi CANCELADA porque existem erros no codigo.
  pause
  exit /b 1
)
echo [2/3] Preparando commit...
git add .
set MSG=Atualizacao Via Verde %date% %time%
git commit -m "%MSG%"
if errorlevel 1 (
  echo Nenhuma alteracao nova para publicar.
  pause
  exit /b 0
)
echo [3/3] Enviando para o GitHub...
git push
if errorlevel 1 (
  echo Falha ao enviar para o GitHub.
  pause
  exit /b 1
)
echo.
echo Atualizacao enviada. O Render fara o novo deploy automaticamente.
pause
