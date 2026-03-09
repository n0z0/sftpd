package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"net"
	"os"
	"sync" // Tambahkan sync untuk mutex map

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// Map global untuk menyimpan daftar presensi: map[IP_Address] = Nomor_Presensi
var (
	daftarPresensi = make(map[string]string)
	presensiMutex  sync.RWMutex
)

func main() {
	// Konfigurasi server SFTP
	host := "0.0.0.0"
	port := "50123" // Port SFTP

	// Path kunci host
	privateKeyPath := "id_rsa" // Ganti dengan path kunci privat server

	// Generate atau baca kunci privat server
	privateKey, err := generateHostKey(privateKeyPath)
	if err != nil {
		log.Fatalf("Error loading private key: %v", err)
	}

	// Konfigurasi SSH server
	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			// Mendapatkan IP address (tanpa port) dari client
			clientIP, _, err := net.SplitHostPort(c.RemoteAddr().String())
			if err != nil {
				// Fallback jika format bukan host:port (misal IP langsung)
				clientIP = c.RemoteAddr().String()
			}

			// Mengecek apakah username adalah IP address client
			if c.User() != clientIP {
				return nil, fmt.Errorf("username %q tidak cocok dengan IP client %q", c.User(), clientIP)
			}

			passwordInput := string(pass)

			// Gunakan Lock untuk membaca/menulis ke map secara aman dari banyak koneksi bersamaan
			presensiMutex.Lock()
			defer presensiMutex.Unlock()

			// Cek apakah IP ini sudah pernah terdaftar login sebelumnya
			registeredPassword, exists := daftarPresensi[clientIP]

			if !exists {
				// ---------------------------------------------------------
				// KONDISI 1: IP INI BARU PERTAMA KALI MASUK (BELUM TERDAFTAR)
				// ---------------------------------------------------------
				// Daftarkan IP ini ke memori beserta "password" yang dia masukkan
				// (yang kita asumsikan ini adalah nomor presensinya yang pertama kali ia ketikkan)
				daftarPresensi[clientIP] = passwordInput
				log.Printf("REGISTRASI BARU: IP %s terdaftar dengan presensi/password %q", clientIP, passwordInput)
				return nil, nil // Sukses login (mendaftar otomatis)

			} else {
				// ---------------------------------------------------------
				// KONDISI 2: IP INI SUDAH TERDAFTAR SEBELUMNYA
				// ---------------------------------------------------------
				// Cek apakah password yang dia masukkan sekarang SAMA dengan password saat dia pertama kali masuk
				if passwordInput == registeredPassword {
					log.Printf("LOGIN SUKSES: IP %s menggunakan presensi %q", clientIP, passwordInput)
					return nil, nil // Sukses login
				}

				// Jika password berbeda dengan yang didaftarkan pertama kali
				log.Printf("LOGIN GAGAL: IP %s memasukkan presensi %q, seharusnya %q", clientIP, passwordInput, registeredPassword)
				return nil, fmt.Errorf("password/presensi salah untuk IP %q", clientIP)
			}
		},
	}
	config.AddHostKey(privateKey)

	// Buka socket server
	listener, err := net.Listen("tcp", net.JoinHostPort(host, port))
	if err != nil {
		log.Fatalf("Failed to listen on %s:%s: %v", host, port, err)
	}
	defer listener.Close()

	log.Printf("SFTP server listening on %s:%s", host, port)

	// Loop untuk menerima koneksi
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		go handleConn(conn, config)
	}
}

func handleConn(conn net.Conn, config *ssh.ServerConfig) {
	// Jalankan handshake SSH
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		log.Printf("Failed to handshake: %v", err)
		return
	}
	defer sshConn.Close()

	log.Printf("New SSH connection from %s (%s)", sshConn.RemoteAddr(), sshConn.ClientVersion())

	// Handle semua request channel
	go ssh.DiscardRequests(reqs)

	// Handle channel SFTP
	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unsupported channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Printf("Could not accept channel: %v", err)
			continue
		}

		go func() {
			for req := range requests {
				if req.Type == "subsystem" && string(req.Payload[4:]) == "sftp" {
					req.Reply(true, nil)
					handleSFTP(channel)
				} else {
					req.Reply(false, nil)
				}
			}
		}()
	}
}

func handleSFTP(channel ssh.Channel) {
	// Buat server SFTP
	server, err := sftp.NewServer(channel)
	if err != nil {
		log.Printf("Failed to create SFTP server: %v", err)
		return
	}
	defer server.Close()

	log.Println("SFTP session started")
	if err := server.Serve(); err == nil {
		log.Println("SFTP session closed")
	} else {
		log.Printf("SFTP server completed with error: %v", err)
	}
}

func generateHostKey(privateKeyPath string) (ssh.Signer, error) {
	// Periksa apakah kunci privat sudah ada
	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		// Jika tidak ada, generate kunci baru
		log.Printf("Generating new host key at %s", privateKeyPath)

		// Generate kunci RSA
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, fmt.Errorf("failed to generate RSA key: %w", err)
		}

		// Encode kunci privat ke format PEM
		privateKeyFile, err := os.Create(privateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create private key file: %w", err)
		}
		defer privateKeyFile.Close()

		err = pem.Encode(privateKeyFile, &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to encode private key: %w", err)
		}

		// Set permission file ke mode aman
		if err := privateKeyFile.Chmod(0600); err != nil {
			return nil, fmt.Errorf("failed to set permissions on key file: %w", err)
		}
	}

	// Baca kunci privat dari file
	privateKeyBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	// Parse kunci privat
	signer, err := ssh.ParsePrivateKey(privateKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return signer, nil
}
