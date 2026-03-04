if ($ExecutionContext.SessionState.LanguageMode -eq "ConstrainedLanguage" -or
    (Get-ExecutionPolicy) -notin @("Unrestricted","Bypass","RemoteSigned")) {
    $ScriptPath = $MyInvocation.MyCommand.Path
    Start-Process powershell.exe -ArgumentList "-ExecutionPolicy Bypass -File `"$ScriptPath`"" -Wait
    exit
}

$Url = "https://github.com/ANP-Automation-Projects/find-it/releases/download/${env:VersionNumber}/main.exe"
$Destination = "$env:TEMP\findit.exe"

try {
    Write-Host "Adding Defender exclusion for TEMP..." -ForegroundColor Yellow
    Add-MpPreference -ExclusionPath $env:TEMP

    Write-Host "Downloading findit.exe..." -ForegroundColor Cyan
    Invoke-WebRequest -Uri $Url -OutFile $Destination -UseBasicParsing
    Write-Host "Download complete: $Destination" -ForegroundColor Green

    Write-Host "Launching findit.exe..." -ForegroundColor Cyan
    & $Destination --type file --ext pst --name "*" --target "*"

    if ($LASTEXITCODE -ne 0) {
        Write-Error "findit.exe exited with code $LASTEXITCODE"
    }

} catch {
    Write-Error "Failed: $_"
} finally {
    Write-Host "Removing Defender exclusion..." -ForegroundColor Yellow
    Remove-MpPreference -ExclusionPath $env:TEMP

    # Clean up the binary when done
    if (Test-Path $Destination) { Remove-Item $Destination -Force }
}