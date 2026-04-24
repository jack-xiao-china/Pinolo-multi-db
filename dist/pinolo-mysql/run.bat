@echo off
REM PINOLO MySQL Logic Bug Detector
REM Usage: run.bat <config_file>

if "%1"=="" (
    echo Usage: run.bat ^<config_file^>
    echo Example: run.bat task_template.json
    exit /b 1
)

impomysql.exe task %1
echo.
echo Results saved to: output\mysql\task-*\
echo Check result.json for statistics
echo Check bugs\ directory for detected logical bugs