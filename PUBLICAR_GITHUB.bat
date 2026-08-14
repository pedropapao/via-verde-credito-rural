@echo off
setlocal
cd /d "%~dp0"
where git >nul 2>nul
if errorlevel 1 (
  echo Git nao foi encontrado. Instale o Git for Windows primeiro.
  pause
  exit /b 1
)
set /p REPO=COLE A URL DO REPOSITORIO GITHUB: 
if "%REPO%"=="" exit /b 1
if not exist .git git init
git add .
git commit -m "Via Verde Credito Rural web v2.0"
git branch -M main
git remote remove origin >nul 2>nul
git remote add origin %REPO%
git push -u origin main
if errorlevel 1 (
  echo.
  echo O GitHub recusou o envio. Verifique login/permissao do Git.
  pause
  exit /b 1
)
echo.
echo Codigo publicado no GitHub.
pause
