# Run manually as the Windows user who performed the E2E initialization.
# The secret remains encrypted on disk; do not capture this command in logs.
Add-Type -AssemblyName System.Security
$taskCredentialPath = Join-Path $PSScriptRoot '../workspace/local-admin.dpapi'
$taskBytes = [System.IO.File]::ReadAllBytes($taskCredentialPath)
$taskPlain = [System.Security.Cryptography.ProtectedData]::Unprotect($taskBytes, $null, [System.Security.Cryptography.DataProtectionScope]::CurrentUser)
Write-Host 'Username: admin'
Write-Host ('Password: ' + [System.Text.Encoding]::UTF8.GetString($taskPlain))
[Array]::Clear($taskPlain, 0, $taskPlain.Length)
