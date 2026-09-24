@echo off
setlocal
cd /d "%~dp0"

echo ============================================================
echo   ViaVerdeCAR - Compilar e publicar atualizacao local
echo ============================================================
echo.
echo Este processo usa o seu proprio Windows e NAO usa minutos
echo do GitHub Actions.
echo.

powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0PUBLICAR_ATUALIZACAO_LOCAL.ps1" -Build
set EXITCODE=%ERRORLEVEL%

echo.
if not "%EXITCODE%"=="0" (
  echo A publicacao NAO foi concluida.
) else (
  echo Processo finalizado.
)
echo.
pause
exit /b %EXITCODE%
