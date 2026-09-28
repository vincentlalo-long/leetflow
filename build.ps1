$ErrorActionPreference = "Stop"

$goBin = go env GOPATH | Join-Path -ChildPath "bin"
$dest = Join-Path $goBin "leet.exe"

Write-Host "Building leet CLI..."
go build -o $dest ./cmd/leet

if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host "Installed to $dest"
Write-Host "Run 'leet --version' from any directory to verify."
Write-Host "Build the terminal UI separately with: go build -o leet-tui.exe ./cmd/leet-tui"
