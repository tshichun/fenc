@echo off

set file=%1

if "%file%" == "" (
    echo unspecified file
) else (
    fenc.exe dec %file%
) 

echo.
pause
exit