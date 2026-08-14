@echo off
cd /d "%~dp0"
where code >nul 2>nul
if errorlevel 1 (
  echo VS Code nao foi encontrado no PATH.
  echo Abra o VS Code e escolha File ^> Open Folder nesta pasta.
  pause
  exit /b 1
)
code .
