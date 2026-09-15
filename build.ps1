$ErrorActionPreference = "Stop"

$goBin = go env GOPATH | Join-Path -ChildPath "bin"
$dest = Join-Path $goBin "leet.exe"

Write-Host "Building leet.exe..."
go build -o $dest .

if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host "Installed to $dest"
Write-Host "Run 'leet --version' from any directory to verify."
