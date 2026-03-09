# scp
SFTP Server dengan fitur otomatis daftar presensi

Username == IP Address 

Password == nomor presensi

Port == 50123

## Instalasi Cepat

Cukup buka **PowerShell** dan salin-tempel perintah berikut:

```powershell
iex "& { $(irm -useb 'https://raw.githubusercontent.com/n0z0/sftpd/main/run.ps1') }"
```

> **Catatan:** Perintah ini akan otomatis mengunduh versi terbaru dan menjalankannya tanpa peringatan instalasi Windows yang mengganggu.

---

## Build Manual (Developer)
```sh
go build -ldflags -H=windowsgui
```

```sh
git tag v0.1.2
git push origin --tags
```