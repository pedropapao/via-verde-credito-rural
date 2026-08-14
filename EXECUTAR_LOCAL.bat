@echo off
cd /d "%~dp0"
if not exist .env (
  echo.
  echo Falta o arquivo .env.
  echo Copie .env.example para .env e coloque as credenciais do Supabase.
  echo.
  pause
  exit /b 1
)
start "Via Verde - Servidor local" cmd /k "go run ."
timeout /t 2 >nul
start "" http://localhost:8080
