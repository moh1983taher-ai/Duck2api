param(
    [string]$Model = "gpt-6-luna",
    [string]$BaseUrl = "http://127.0.0.1:18083"
)

$ErrorActionPreference = "Stop"
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
        },
        @{
            type = "mcp"
            server_label = "shamela"
            server_url = "https://mcp.shamela.ws"
            allowed_tools = @{ read_only = $true }
            require_approval = "never"
        }
    )
    input = "ابحث في الجامع والمكتبة الشاملة عن دلالة الأمر الوارد بعد الحظر، واعرض أقوال الأصوليين وأدلتهم ومراجعهم."
} | ConvertTo-Json -Depth 30

$response = Invoke-RestMethod -Method Post -Uri "$BaseUrl/v1/responses" -Headers $headers -Body $body -TimeoutSec 480
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
