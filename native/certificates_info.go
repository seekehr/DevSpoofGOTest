package native

import (
	"crypto/sha1" // For SHA-1 thumbprint
	"crypto/x509"
	"encoding/hex" // For converting hash to hex string
	"fmt"
	"log"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func GetCertificatesFromRegistry() ([]string, error) {
	certRegPath := `SOFTWARE\Microsoft\SystemCertificates\ROOT\Certificates`
	var certSummaries []string

	k, err := registry.OpenKey(registry.LOCAL_MACHINE, certRegPath, registry.READ)
	if err != nil {
		return nil, fmt.Errorf("failed to open registry key %s: %w", certRegPath, err)
	}
	defer k.Close()

	subKeyNames, err := k.ReadSubKeyNames(-1)
	if err != nil {
		return nil, fmt.Errorf("failed to read subkey names under %s: %w", certRegPath, err)
	}

	if len(subKeyNames) == 0 {
		log.Printf("Info: No subkeys found under HKLM\\%s. Certificates might be stored differently or no ROOT certificates are present.", certRegPath)
		return certSummaries, nil
	}

	for _, subKeyName := range subKeyNames {
		subK, err := registry.OpenKey(k, subKeyName, registry.READ)
		if err != nil {
			log.Printf("Warning: Failed to open subkey %s\\%s: %v", certRegPath, subKeyName, err)
			continue
		}
		defer subK.Close() // Close immediately after iteration

		var certBlob []byte
		var foundBlob bool

		// 1. Prioritize reading the "Blob" value
		certBlob, _, err = subK.GetBinaryValue("Blob")
		if err == nil {
			foundBlob = true
		} else if err != registry.ErrNotExist {
			log.Printf("Debug: Could not read 'Blob' value from %s\\%s: %v", certRegPath, subKeyName, err)
		}

		// 2. If "Blob" not found or error, try to find other binary values
		if !foundBlob {
			valNames, _ := subK.ReadValueNames(-1)
			for _, valName := range valNames {
				tempBlob, _, readErr := subK.GetBinaryValue(valName)
				if readErr == nil {
					certBlob = tempBlob
					foundBlob = true
					log.Printf("Info: Found binary value '%s' in %s\\%s, attempting to parse as certificate.\n", valName, certRegPath, subKeyName)
					break
				} else if readErr != registry.ErrNotExist {
					log.Printf("Debug: Could not read '%s' as binary from %s\\%s: %v", valName, certRegPath, subKeyName, readErr)
				}
			}
		}

		if !foundBlob || len(certBlob) == 0 {
			log.Printf("Warning: No suitable binary blob found in subkey %s\\%s.", certRegPath, subKeyName)
			continue
		}

		var cert *x509.Certificate
		var currentParseError error

		// --- NEW STRATEGY: Find the DER certificate within the blob ---
		// We'll try to find the start of a DER certificate (30 82 or 30 81)
		// This handles cases where the blob has a header before the actual certificate.
		derStart := -1
		for i := 0; i < len(certBlob)-1; i++ {
			if certBlob[i] == 0x30 && (certBlob[i+1] == 0x82 || certBlob[i+1] == 0x81) {
				derStart = i
				break
			}
		}

		if derStart != -1 {
			// Slice the blob to start at the DER certificate
			potentialCertBytes := certBlob[derStart:]

			// Try parsing as a single DER certificate first
			parsedCert, parseErr := x509.ParseCertificate(potentialCertBytes)
			if parseErr == nil {
				cert = parsedCert
			} else {
				currentParseError = parseErr // Store the error from ParseCertificate

				// If that fails, try parsing as a sequence of DER certificates
				parsedCerts, parseMultiErr := x509.ParseCertificates(potentialCertBytes)
				if parseMultiErr == nil && len(parsedCerts) > 0 {
					cert = parsedCerts[0]
					if len(parsedCerts) > 1 {
						log.Printf("Info: Found %d certificates in blob from %s\\%s. Processing the first one.", certRegPath, subKeyName, len(parsedCerts))
					}
				} else {
					log.Printf("Warning: Failed to parse X.509 certificate from %s\\%s blob (subkey: %s). Initial error (after offset attempt): %v. Multi-cert error (after offset attempt): %v. Blob Hex: %s",
						certRegPath, subKeyName, subKeyName, currentParseError, parseMultiErr, hex.EncodeToString(certBlob))
					continue
				}
			}
		} else {
			// No DER certificate start found
			log.Printf("Warning: No DER certificate header (3081 or 3082) found in blob from %s\\%s (subkey: %s). Blob Hex: %s",
				certRegPath, subKeyName, subKeyName, hex.EncodeToString(certBlob))
			continue
		}
		// --- END NEW STRATEGY ---

		// If a certificate was successfully parsed
		if cert != nil {
			hasher := sha1.New()
			hasher.Write(cert.Raw)
			thumbprint := hex.EncodeToString(hasher.Sum(nil))

			summary := fmt.Sprintf(
				"Subject: %s | Issuer: %s | Valid From: %s | Valid To: %s | Thumbprint (SHA1): %s",
				cert.Subject.CommonName,
				cert.Issuer.CommonName,
				cert.NotBefore.Format("2006-01-02"),
				cert.NotAfter.Format("2006-01-02"),
				strings.ToUpper(thumbprint),
			)
			certSummaries = append(certSummaries, summary)
		}
	}

	return certSummaries, nil
}
