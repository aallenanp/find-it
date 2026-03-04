# Self-elevate with execution policy bypass if not already running unrestricted
if ($ExecutionContext.SessionState.LanguageMode -eq "ConstrainedLanguage" -or
    (Get-ExecutionPolicy) -notin @("Unrestricted","Bypass","RemoteSigned")) {
    $ScriptPath = $MyInvocation.MyCommand.Path
    Start-Process powershell.exe -ArgumentList "-ExecutionPolicy Bypass -File `"$ScriptPath`"" -Wait
    exit
}

# Download and Run findit.exe
$Url = "https://github.com/ANP-Automation-Projects/find-it/releases/download/v1.0.1/main.exe"
$Destination = "$env:TEMP\findit.exe"

Write-Host "Downloading findit.exe..." -ForegroundColor Cyan

try {
    Invoke-WebRequest -Uri $Url -OutFile $Destination -UseBasicParsing
    Write-Host "Download complete: $Destination" -ForegroundColor Green
} catch {
    Write-Error "Failed to download file: $_"
    exit 1
}

Write-Host "Launching findit.exe..." -ForegroundColor Cyan

try {
    & $Destination --type file --ext pst --name "*" --target "*"
    if ($LASTEXITCODE -ne 0) {
        Write-Error "findit.exe exited with code $LASTEXITCODE"
        exit $LASTEXITCODE
    }
    Write-Host "Process completed." -ForegroundColor Green
} catch {
    Write-Error "Failed to run executable: $_"
    exit 1
}