if ($ExecutionContext.SessionState.LanguageMode -eq "ConstrainedLanguage" -or
    (Get-ExecutionPolicy) -notin @("Unrestricted","Bypass","RemoteSigned")) {
    $ScriptPath = $MyInvocation.MyCommand.Path
    Start-Process powershell.exe -ArgumentList "-ExecutionPolicy Bypass -File `"$ScriptPath`"" -Wait
    exit
}

$Url = "https://github.com/ANP-Automation-Projects/find-it/releases/download/${env:VersionNumber}/main.exe"
$Destination = "$env:TEMP\findit.exe"

$type   = if ($env:SearchType)     { $env:SearchType }     else { "file" }
$ext    = if ($env:FileExt)      { $env:FileExt }      else { "txt" }
$name   = if ($env:SearchString) { $env:SearchString } else { "*" }
$target = if ($env:SearchTarget) { $env:SearchTarget } else { "*" }

try {
    Write-Host "Adding Defender exclusion for TEMP..." -ForegroundColor Yellow
    Add-MpPreference -ExclusionPath $env:TEMP

    Write-Host "Downloading findit.exe..." -ForegroundColor Cyan
    Invoke-WebRequest -Uri $Url -OutFile $Destination -UseBasicParsing
    Write-Host "Download complete: $Destination" -ForegroundColor Green

    Write-Host "Launching findit.exe..." -ForegroundColor Cyan
    & $Destination --type "$type" --ext "$ext" --name "$name" --target "$target"

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