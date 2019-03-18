@echo off

set path=%1

if "%path%" == "" (
    echo unspecified path
) else (
    fenc.exe enc %path%
) 

echo.
pause
exit