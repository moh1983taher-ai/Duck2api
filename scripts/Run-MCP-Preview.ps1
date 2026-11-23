$ErrorActionPreference = "Stop"

$previewPort = 18083
$existing = Get-NetTCPConnection -LocalPort $previewPort -State Listen -ErrorAction SilentlyContinue
if ($existing) {
    throw "Port $previewPort is already in use. Stop the preview service that owns it, then retry. The installed service on port 18080 is not touched."
}

$packageRoot = Split-Path -Parent $PSScriptRoot
$binary = Join-Path $packageRoot "duck2api.exe"
if (-not (Test-Path $binary)) {
    throw "duck2api.exe was not found beside this script."
}

$env:SERVER_HOST = "127.0.0.1"
$env:SERVER_PORT = "$previewPort"
if ([string]::IsNullOrWhiteSpace($env:MCP_SERVERS)) {
    $env:MCP_SERVERS = '[{"server_label":"aljam3","server_url":"https://aljam3.com/mcp","allow_read_only_tools":true,"require_approval":"never"},{"server_label":"shamela","server_url":"https://mcp.shamela.ws","allow_read_only_tools":true,"require_approval":"never"}]'
}
if ([string]::IsNullOrWhiteSpace($env:MCP_MAX_TOOL_CALLS)) {
    $env:MCP_MAX_TOOL_CALLS = "64"
}

Push-Location $packageRoot
try {
    Write-Host "Starting isolated Duck2API MCP preview on http://127.0.0.1:$previewPort"
    Write-Host "The existing service on port 18080 is not modified."
    & $binary
}
finally {
    Pop-Location
}
