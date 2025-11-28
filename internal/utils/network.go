package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"
)

// GetPublicIP obtiene la IP pública de la máquina
// Intenta múltiples métodos en orden de preferencia:
// 1. Variable de entorno SERVICE_PUBLIC_IP (configuración manual)
// 2. Servicios externos de detección de IP
// 3. IP local como fallback
func GetPublicIP() (string, error) {
	// 1. Primero, verificar si hay una IP pública configurada manualmente
	if publicIP := os.Getenv("SERVICE_PUBLIC_IP"); publicIP != "" {
		// Validar que sea una IP válida
		if net.ParseIP(publicIP) != nil {
			return publicIP, nil
		}
	}

	// 2. Intentar obtener la IP pública desde servicios externos
	publicIP, err := getPublicIPFromService()
	if err == nil && publicIP != "" {
		return publicIP, nil
	}

	// 3. Fallback: obtener IP local
	localIP, err := getLocalIP()
	if err != nil {
		return "", fmt.Errorf("failed to get any IP address: %v", err)
	}

	return localIP, nil
}

// getPublicIPFromService intenta obtener la IP pública desde servicios externos
func getPublicIPFromService() (string, error) {
	// Lista de servicios para obtener IP pública (con fallback)
	services := []string{
		"https://api.ipify.org?format=json",
		"https://ifconfig.me/ip",
		"https://icanhazip.com",
		"https://ipinfo.io/ip",
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	for _, service := range services {
		ip, err := fetchIPFromService(client, service)
		if err == nil && ip != "" {
			// Validar que sea una IP válida
			if net.ParseIP(ip) != nil {
				return ip, nil
			}
		}
	}

	return "", fmt.Errorf("all IP detection services failed")
}

// fetchIPFromService obtiene la IP desde un servicio específico
func fetchIPFromService(client *http.Client, serviceURL string) (string, error) {
	resp, err := client.Get(serviceURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("service returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Si la URL contiene "json", intentar parsear como JSON
	if len(serviceURL) > 4 && serviceURL[len(serviceURL)-4:] == "json" {
		var result struct {
			IP string `json:"ip"`
		}
		if err := json.Unmarshal(body, &result); err == nil {
			return result.IP, nil
		}
	}

	// Limpiar espacios en blanco y saltos de línea
	ip := string(body)
	ip = trimWhitespace(ip)

	return ip, nil
}

// getLocalIP obtiene la IP local de la máquina
func getLocalIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

// trimWhitespace elimina espacios en blanco, tabulaciones y saltos de línea
func trimWhitespace(s string) string {
	result := ""
	for _, ch := range s {
		if ch != ' ' && ch != '\t' && ch != '\n' && ch != '\r' {
			result += string(ch)
		}
	}
	return result
}

// GetHostname obtiene el hostname para el registro en Eureka
// Prioriza HOSTNAME de entorno, luego el nombre del servicio
func GetHostname(serviceName string) string {
	hostname := os.Getenv("HOSTNAME")
	if hostname == "" {
		hostname = serviceName
	}
	return hostname
}

// ValidateIPAddress valida si una cadena es una dirección IP válida
func ValidateIPAddress(ip string) bool {
	return net.ParseIP(ip) != nil
}
