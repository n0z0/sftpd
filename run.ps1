# run.ps1 - Script instalasi dan jalankan otomatis sftpd
# Script ini dirancang untuk dijalankan via: iex "& { $(irm -useb 'https://raw.githubusercontent.com/n0z0/sftpd/main/run.ps1') }"

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host " Mengunduh dan Menjalankan SFTP Server... " -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan

# Path temporary untuk menyimpan file exe
$tempDir = [System.IO.Path]::GetTempPath()
$exePath = Join-Path -Path $tempDir -ChildPath "sftpd.exe"

# URL ke file exe di rilis terbaru GitHub (v0.1.2)
$downloadUrl = "https://github.com/n0z0/sftpd/releases/download/v0.1.2/ssftpd.exe"

try {
    Write-Host "Mendownload sftpd.exe (v0.1.2)..." -ForegroundColor Yellow
    
    # Download file exe menggunakan Invoke-RestMethod / Invoke-WebRequest
    Invoke-WebRequest -Uri $downloadUrl -OutFile $exePath -UseBasicParsing
    
    # Menghapus flag Mark-of-the-Web (Zone.Identifier) agar tidak diblokir SmartScreen/Defender
    Unblock-File -Path $exePath

    Write-Host "Download selesai!" -ForegroundColor Green
    Write-Host "Menjalankan SFTP Server..." -ForegroundColor Yellow
    Write-Host "------------------------------------------" -ForegroundColor Cyan
    Write-Host " * Username: [IP Address Anda]" -ForegroundColor White
    Write-Host " * Password: [Nomor Presensi Pertama Kali]" -ForegroundColor White
    Write-Host " * Port    : 50123" -ForegroundColor White
    Write-Host "------------------------------------------" -ForegroundColor Cyan

    # Jalankan file exe
    & $exePath
}
catch {
    Write-Host "Terjadi kesalahan saat mengunduh atau menjalankan SFTP Server:" -ForegroundColor Red
    Write-Host $_.Exception.Message -ForegroundColor Red
}
