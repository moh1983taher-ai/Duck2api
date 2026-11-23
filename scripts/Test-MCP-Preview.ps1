param(
    [string]$Model = "gpt-6-luna",
    [string]$BaseUrl = "http://127.0.0.1:18083",
    [switch]$IncludeShamela
)

$ErrorActionPreference = "Stop"
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$headers = @{ "Content-Type" = "application/json" }
if (-not [string]::IsNullOrWhiteSpace($env:Authorization)) {
    $headers["Authorization"] = "Bearer $($env:Authorization)"
}

$body = @{
    model = $Model
    tool_choice = "required"
    tools = @(
        @{
            type = "mcp"
            server_label = "aljam3"
            server_url = "https://aljam3.com/mcp"
            allowed_tools = @{ read_only = $true }
            require_approval = "never"
        }
    )
    input = [System.Text.Encoding]::UTF8.GetString([Convert]::FromBase64String("2KfYqNit2Ksg2YHZiiDYp9mE2KzYp9mF2Lkg2YjYp9mE2YXZg9iq2KjYqSDYp9mE2LTYp9mF2YTYqSDYudmGINiv2YTYp9mE2Kkg2KfZhNij2YXYsSDYp9mE2YjYp9ix2K8g2KjYudivINin2YTYrdi42LHYjCDZiNin2LnYsdi2INij2YLZiNin2YQg2KfZhNij2LXZiNmE2YrZitmGINmI2KPYr9mE2KrZh9mFINmI2YXYsdin2KzYudmH2YUu"))
} | ConvertTo-Json -Depth 30

if ($IncludeShamela) {
    $payload = $body | ConvertFrom-Json
    $payload.tools += @{
        type = "mcp"
        server_label = "shamela"
        server_url = "https://mcp.shamela.ws"
        allowed_tools = @{ read_only = $true }
        require_approval = "never"
    }
    $body = $payload | ConvertTo-Json -Depth 30
}

$requestBytes = [System.Text.Encoding]::UTF8.GetBytes($body)
try {
    $response = Invoke-RestMethod -Method Post -Uri "$BaseUrl/v1/responses" -Headers $headers -ContentType "application/json; charset=utf-8" -Body $requestBytes -TimeoutSec 480
}
catch {
    $details = $_.ErrorDetails.Message
    if ($details -match "429|Too Many Requests") {
        throw "Duck.ai returned HTTP 429. The MCP test stopped without retrying. Wait before the next attempt; inspect the preview server log to see whether a tool call occurred."
    }
    throw
}
$calls = @($response.output | Where-Object { $_.type -eq "mcp_call" })
$completed = @($calls | Where-Object { $_.status -eq "completed" })
if ($completed.Count -lt 1) {
    throw "The model did not complete any MCP tool call."
}
if ([string]::IsNullOrWhiteSpace($response.output_text)) {
    throw "The Responses API did not return a final assistant answer."
}

Write-Host "Model: $($response.model)"
Write-Host "MCP calls: $($calls.Count); completed: $($completed.Count)"
Write-Host "Final answer characters: $($response.output_text.Length)"
Write-Output $response.output_text
